package stub

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/reflector"
	"github.com/fasthttp/websocket"
)

type (
	Header              map[string]string
	RPCContainerHandler func(ctx context.Context, c kit.IncomingRPCContainer)
	RPCMessageHandler   func(ctx context.Context, msg kit.Message, hdr Header, err error)
)

type RPCPreflightHandler func(req *WebsocketRequest)

type WebsocketCtx struct {
	cfg wsConfig
	r   *reflector.Reflector
	l   kit.Logger

	pendingMtx   sync.Mutex
	pending      map[string]chan kit.IncomingRPCContainer
	lastActivity uint32
	disconnect   bool

	// fasthttp entities
	url  string
	cMtx sync.Mutex
	c    *websocket.Conn
}

func (wCtx *WebsocketCtx) Connect(ctx context.Context, path string) error {
	path = strings.TrimLeft(path, "/")
	if path != "" {
		wCtx.url = fmt.Sprintf("%s/%s", wCtx.url, path)
	}

	return wCtx.connect(ctx)
}

func (wCtx *WebsocketCtx) connect(ctx context.Context) error {
	wCtx.l.Debugf("connect: %s", wCtx.url)

	d := wCtx.cfg.dialerBuilder()
	if f := wCtx.cfg.preDial; f != nil {
		f(d)
	}
	c, rsp, err := d.DialContext(ctx, wCtx.url, wCtx.cfg.upgradeHdr)
	if err != nil {
		return err
	}
	_ = rsp.Body.Close()

	wCtx.setActivity()
	c.SetPongHandler(
		func(appData string) error {
			wCtx.l.Debugf("websocket pong received")
			wCtx.setActivity()

			return nil
		},
	)

	wCtx.c = c

	// run receiver & watchdog in the background
	go wCtx.receiver(c) //nolint:contextcheck
	go wCtx.watchdog()  //nolint:contextcheck

	if f := wCtx.cfg.onConnect; f != nil {
		f(wCtx)
	}

	return nil
}

func (wCtx *WebsocketCtx) Disconnect() error {
	wCtx.disconnect = true

	return wCtx.c.Close()
}

func (wCtx *WebsocketCtx) setActivity() {
	atomic.StoreUint32(&wCtx.lastActivity, uint32(utils.TimeUnix()))
}

func (wCtx *WebsocketCtx) getActivity() int64 {
	return int64(atomic.LoadUint32(&wCtx.lastActivity))
}

func (wCtx *WebsocketCtx) watchdog() {
	wCtx.l.Debugf("watchdog started: %s", wCtx.c.LocalAddr().String())

	t := time.NewTicker(wCtx.cfg.pingTime)
	d := int64(wCtx.cfg.pingTime/time.Second) * 2
	for range t.C {
		if wCtx.disconnect {
			wCtx.l.Debugf("going to disconnect: %s", wCtx.c.LocalAddr().String())

			_ = wCtx.c.Close()

			return
		}

		if utils.TimeUnix()-wCtx.getActivity() <= d {
			wCtx.cMtx.Lock()
			_ = wCtx.c.WriteControl(websocket.PingMessage, nil, time.Now().Add(wCtx.cfg.writeTimeout))
			wCtx.cMtx.Unlock()
			wCtx.l.Debugf("websocket ping sent")

			continue
		}

		if !wCtx.cfg.autoReconnect {
			return
		}

		wCtx.l.Debugf("inactivity detected, reconnecting: %s", wCtx.c.LocalAddr().String())
		_ = wCtx.c.Close()

		ctx, cf := context.WithTimeout(context.Background(), wCtx.cfg.dialTimeout)
		err := wCtx.connect(ctx)
		cf()
		if err != nil {
			wCtx.l.Debugf("failed to reconnect: %s", err)

			continue
		}

		return
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

		rpcIn := wCtx.cfg.rpcInFactory()
		err = rpcIn.Unmarshal(p)
		if err != nil {
			wCtx.l.Debugf("received unexpected message: %v", err)

			continue
		}

		// if this is a reply message we return it to the pending channel
		wCtx.pendingMtx.Lock()
		ch, ok := wCtx.pending[rpcIn.GetID()]
		wCtx.pendingMtx.Unlock()

		if ok {
			ch <- rpcIn

			continue
		}

		ctx := context.Background()
		if tp := wCtx.cfg.tracePropagator; tp != nil {
			ctx = tp.Extract(ctx, containerTraceCarrier{in: rpcIn})
		}

		h, ok := wCtx.cfg.handlers[rpcIn.GetHdr(wCtx.cfg.predicateKey)]
		if !ok {
			h = wCtx.cfg.defaultHandler
		}

		if h == nil {
			rpcIn.Release()

			continue
		}

		wCtx.cfg.ratelimitChan <- struct{}{}
		wCtx.cfg.handlersWG.Add(1)
		go func(ctx context.Context, rpcIn kit.IncomingRPCContainer) {
			defer wCtx.recoverPanic()

			h(ctx, rpcIn)
			<-wCtx.cfg.ratelimitChan
			wCtx.cfg.handlersWG.Done()
			rpcIn.Release()
		}(ctx, rpcIn)
	}
}

func (wCtx *WebsocketCtx) recoverPanic() {
	if r := recover(); r != nil {
		wCtx.l.Errorf("panic recovered: %v", r)

		if wCtx.cfg.panicRecoverFunc != nil {
			wCtx.cfg.panicRecoverFunc(r)
		}
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

// NetConn returns the underlying net.Conn, ONLY for advanced use cases
func (wCtx *WebsocketCtx) NetConn() net.Conn {
	return wCtx.c.NetConn()
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
	// run preflights
	for _, pre := range wCtx.cfg.preflights {
		pre(&req)
	}

	outC := wCtx.cfg.rpcOutFactory()
	id := utils.RandomDigit(10)
	outC.InjectMessage(req.ReqMsg)
	outC.SetHdr(wCtx.cfg.predicateKey, req.Predicate)
	if tp := wCtx.cfg.tracePropagator; tp != nil {
		tp.Inject(ctx, containerTraceCarrier{out: outC})
	}
	for k, v := range req.ReqHdr {
		outC.SetHdr(k, v)
	}
	outC.SetID(id)

	reqData, err := outC.Marshal()
	if err != nil {
		return err
	}

	wCtx.cMtx.Lock()
	err = wCtx.c.WriteMessage(req.MessageType, reqData)
	wCtx.cMtx.Unlock()
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
	wCtx.pendingMtx.Lock()
	wCtx.pending[id] = resCh
	wCtx.pendingMtx.Unlock()

	select {
	case c := <-resCh:
		err := c.ExtractMessage(res)
		cb(ctx, res, c.GetHdrMap(), err)

	case <-ctx.Done():
	}

	wCtx.pendingMtx.Lock()
	delete(wCtx.pending, id)
	wCtx.pendingMtx.Unlock()
}

type containerTraceCarrier struct {
	out kit.OutgoingRPCContainer
	in  kit.IncomingRPCContainer
}

func (c containerTraceCarrier) Get(key string) string {
	return c.in.GetHdr(key)
}

func (c containerTraceCarrier) Set(key string, value string) {
	c.out.SetHdr(key, value)
}
