package stub

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/reflector"
	"github.com/fasthttp/websocket"
)

type WebsocketCtx struct {
	cfg     wsConfig
	err     *Error
	r       *reflector.Reflector
	l       kit.Logger
	dumpReq io.Writer
	dumpRes io.Writer

	pendingL     sync.Mutex
	pending      map[string]chan kit.IncomingRPCContainer
	lastActivity int64
	disconnect   bool
	path         string

	// fasthttp entities
	url string
	c   *websocket.Conn
}

func (wCtx *WebsocketCtx) Connect(ctx context.Context, path string) error {
	path = strings.TrimLeft(path, "/")
	wCtx.path = path

	return wCtx.connect(ctx)
}

func (wCtx *WebsocketCtx) connect(ctx context.Context) error {
	url := wCtx.url
	if wCtx.path != "" {
		url = fmt.Sprintf("%s/%s", wCtx.url, wCtx.path)
	}
	wCtx.l.Debugf("connect: %s", url)

	d := wCtx.cfg.dialerBuilder()
	c, _, err := d.DialContext(ctx, url, wCtx.cfg.upgradeHdr)
	if err != nil {
		return err
	}

	wCtx.setActivity()
	c.SetPongHandler(func(appData string) error {
		wCtx.l.Debug("pong received")
		wCtx.setActivity()

		return nil
	})

	wCtx.c = c

	// run receiver & watchdog in background
	go wCtx.receiver(c)
	go wCtx.watchdog()

	if wCtx.cfg.onConnect != nil {
		wCtx.cfg.onConnect(wCtx)
	}

	return nil
}

func (wCtx *WebsocketCtx) Disconnect() error {
	wCtx.disconnect = true

	return wCtx.c.Close()
}

func (wCtx *WebsocketCtx) setActivity() {
	atomic.StoreInt64(&wCtx.lastActivity, utils.TimeUnix())
}

func (wCtx *WebsocketCtx) watchdog() {
	wCtx.l.Debugf("watchdog started: %s", wCtx.c.LocalAddr().String())

	t := time.NewTicker(wCtx.cfg.pingTime)
	d := int64(wCtx.cfg.pingTime/time.Second) * 2
	for {
		select {
		case <-t.C:
			if wCtx.disconnect {
				wCtx.l.Debugf("going to disconnect: %s", wCtx.c.LocalAddr().String())

				_ = wCtx.c.Close()

				return
			}

			if utils.TimeUnix()-wCtx.lastActivity > d {
				if wCtx.cfg.autoReconnect {
					wCtx.l.Debugf("inactivity detected, reconnecting: %s", wCtx.c.LocalAddr().String())

					_ = wCtx.c.Close()

					ctx, cf := context.WithTimeout(context.Background(), wCtx.cfg.dialTimeout)
					err := wCtx.connect(ctx)
					cf()
					if err != nil {
						wCtx.l.Debugf("failed to reconnect: %s", err)

						continue
					}
				}

				return
			}

			_ = wCtx.c.WriteControl(websocket.PingMessage, nil, time.Now().Add(wCtx.cfg.writeTimeout))
			wCtx.l.Debug("ping sent")
		}
	}
}

func (wCtx *WebsocketCtx) receiver(c *websocket.Conn) {
	for {
		_, p, err := c.ReadMessage()
		if err != nil || len(p) == 0 {
			wCtx.l.Debugf("receiver shutdown: %s: %v", c.LocalAddr().String(), err)

			return
		}

		wCtx.setActivity()

		c := wCtx.cfg.rpcInFactory()
		err = c.Unmarshal(p)
		if err != nil {
			wCtx.l.Debugf("received unexpected message: %v", err)

			continue
		}
		wCtx.pendingL.Lock()
		ch, ok := wCtx.pending[c.GetID()]
		wCtx.pendingL.Unlock()

		if ok {
			ch <- c

			continue
		}

		if h, ok := wCtx.cfg.handlers[c.GetHdr(wCtx.cfg.predicateKey)]; ok {
			h(c)
		} else if h = wCtx.cfg.defaultHandler; h != nil {
			h(c)
		}
		c.Release()
	}
}

func (wCtx *WebsocketCtx) TextMessage(
	ctx context.Context, predicate string, req, res kit.Message,
	cb RPCMessageHandler,
) error {
	return wCtx.Do(
		ctx,
		WebsocketRequest{
			Predicate:   predicate,
			MessageType: websocket.TextMessage,
			ReqMsg:      req,
			ResMsg:      res,
			ReqHdr:      nil,
			Callback:    cb,
		},
	)
}

func (wCtx *WebsocketCtx) BinaryMessage(
	ctx context.Context, predicate string, req, res kit.Message,
	cb RPCMessageHandler,
) error {
	return wCtx.Do(
		ctx,
		WebsocketRequest{
			Predicate:   predicate,
			MessageType: websocket.BinaryMessage,
			ReqMsg:      req,
			ResMsg:      res,
			ReqHdr:      nil,
			Callback:    cb,
		},
	)
}

type WebsocketRequest struct {
	Predicate   string
	MessageType int
	ReqMsg      kit.Message
	ResMsg      kit.Message
	ReqHdr      Header
	Callback    RPCMessageHandler
}

func (wCtx *WebsocketCtx) Do(ctx context.Context, req WebsocketRequest) error {
	outC := wCtx.cfg.rpcOutFactory()

	id := utils.RandomDigit(10)
	outC.InjectMessage(req.ReqMsg)
	outC.SetHdr(wCtx.cfg.predicateKey, req.Predicate)
	outC.SetID(id)

	reqData, err := outC.Marshal()
	if err != nil {
		return err
	}

	err = wCtx.c.WriteMessage(req.MessageType, reqData)
	if err != nil {
		return err
	}

	outC.Release()

	go wCtx.waitForMessage(ctx, id, req.ResMsg, req.Callback)

	return nil
}

func (wCtx *WebsocketCtx) waitForMessage(
	ctx context.Context, id string, res kit.Message, cb RPCMessageHandler,
) {
	resCh := make(chan kit.IncomingRPCContainer, 1)
	wCtx.pendingL.Lock()
	wCtx.pending[id] = resCh
	wCtx.pendingL.Unlock()

	select {
	case c := <-resCh:
		err := c.ExtractMessage(res)
		cb(ctx, res, c.GetHdrMap(), err)

	case <-ctx.Done():
	}

	wCtx.pendingL.Lock()
	delete(wCtx.pending, id)
	wCtx.pendingL.Unlock()
}
