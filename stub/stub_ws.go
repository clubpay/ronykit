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

	pendingMtx     sync.Mutex
	pending        map[string]chan kit.IncomingRPCContainer
	lastActivity   uint32
	disconnectChan chan struct{}

	// fasthttp entities
	url  string
	cMtx sync.Mutex
	wg   sync.WaitGroup
	c    *websocket.Conn

	// stats
	writeBytesTotal uint64
	writeBytes      uint64
	readBytesTotal  uint64
	readBytes       uint64
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
		func(_ string) error {
			wCtx.l.Debugf("websocket pong received")
			wCtx.setActivity()

			return nil
		},
	)
	err = c.SetCompressionLevel(wCtx.cfg.compressLevel)
	if err != nil {
		return err
	}

	wCtx.c = c
	wCtx.writeBytes = 0
	wCtx.readBytes = 0

	// run receiver & watchdog in the background
	wCtx.wg.Add(2)
	go wCtx.receiver(c) //nolint:contextcheck
	go wCtx.watchdog(c) //nolint:contextcheck

	if f := wCtx.cfg.onConnect; f != nil {
		f(wCtx)
	}

	return nil
}

func (wCtx *WebsocketCtx) Disconnect() {
	wCtx.disconnectChan <- struct{}{}
	wCtx.wg.Wait()
}

func (wCtx *WebsocketCtx) Reconnect(ctx context.Context) error {
	wCtx.Disconnect()

	return wCtx.connect(ctx)
}

func (wCtx *WebsocketCtx) setActivity() {
	atomic.StoreUint32(&wCtx.lastActivity, uint32(utils.TimeUnix()))
}

func (wCtx *WebsocketCtx) getActivity() int64 {
	return int64(atomic.LoadUint32(&wCtx.lastActivity))
}

func (wCtx *WebsocketCtx) watchdog(c *websocket.Conn) {
	defer wCtx.wg.Done()

	wCtx.l.Debugf("watchdog started: %s", c.LocalAddr().String())

	t := time.NewTicker(wCtx.cfg.pingTime)
	d := int64(wCtx.cfg.pingTime/time.Second) * 2
	for {
		select {
		case <-wCtx.disconnectChan:
			_ = c.Close()

			return

		case <-t.C:
			if utils.TimeUnix()-wCtx.getActivity() <= d {
				wCtx.cMtx.Lock()
				err := c.WriteControl(websocket.PingMessage, nil, time.Now().Add(wCtx.cfg.writeTimeout))
				if err != nil {
					wCtx.l.Errorf("failed to send ping: %v", err)
				}
				wCtx.cMtx.Unlock()
				wCtx.l.Debugf("websocket ping sent")

				continue
			}

			if !wCtx.cfg.autoReconnect {
				return
			}

			wCtx.l.Errorf("inactivity detected, reconnecting: %s", c.LocalAddr().String())
			_ = c.Close()

			ctx, cf := context.WithTimeout(context.Background(), wCtx.cfg.dialTimeout)
			err := wCtx.connect(ctx)
			cf()
			if err != nil {
				wCtx.l.Errorf("failed to reconnect: %s", err)

				continue
			}

			return
		}
	}
}

func (wCtx *WebsocketCtx) receiver(c *websocket.Conn) {
	defer wCtx.wg.Done()

	for {
		_, p, err := c.ReadMessage()
		if err != nil || len(p) == 0 {
			wCtx.l.Debugf("receiver shutdown: %s: %v", c.LocalAddr().String(), err)

			return
		}

		wCtx.readBytesTotal += uint64(len(p))
		wCtx.readBytes += uint64(len(p))
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

		select {
		default:
			wCtx.l.Errorf("ratelimit reached, packet dropped")
		case wCtx.cfg.rateLimitChan <- struct{}{}:
			wCtx.cfg.handlersWG.Add(1)
			go func(ctx context.Context, rpcIn kit.IncomingRPCContainer) {
				defer wCtx.recoverPanic()

				h(ctx, rpcIn)
				<-wCtx.cfg.rateLimitChan
				wCtx.cfg.handlersWG.Done()
				rpcIn.Release()
			}(ctx, rpcIn)
		}
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

type WebsocketStats struct {
	// ReadBytes is the total number of bytes read from the current websocket connection
	ReadBytes uint64
	// ReadBytesTotal is the total number of bytes read since WebsocketCtx creation
	ReadBytesTotal uint64
	// WriteBytes is the total number of bytes written to the current websocket connection
	WriteBytes uint64
	// WriteBytesTotal is the total number of bytes written since WebsocketCtx creation
	WriteBytesTotal uint64
}

func (wCtx *WebsocketCtx) Stats() WebsocketStats {
	wCtx.cMtx.Lock()
	defer wCtx.cMtx.Unlock()

	return WebsocketStats{
		ReadBytes:       wCtx.readBytes,
		ReadBytesTotal:  wCtx.readBytesTotal,
		WriteBytes:      wCtx.writeBytes,
		WriteBytesTotal: wCtx.writeBytesTotal,
	}
}

type WebsocketRequest struct {
	// ID is optional, if you don't set it, a random string will be generated
	ID string
	// Predicate is the routing key for the message, which will be added to the kit.OutgoingRPCContainer
	Predicate string
	// MessageType is the type of the message, either websocket.TextMessage or websocket.BinaryMessage
	MessageType int
	ReqMsg      kit.Message
	// ResMsg is the message that will be used to unmarshal the response.
	// You should pass a pointer to the struct that you want to unmarshal the response into.
	// If Callback is nil, then this field will be ignored.
	ResMsg kit.Message
	// ReqHdr is the headers that will be added to the kit.OutgoingRPCContainer
	ReqHdr Header
	// Callback is the callback that will be called when the response is received.
	// If this is nil, the response will be ignored. However, the response will be caught by
	// the default handler if it is set.
	Callback RPCMessageHandler
}

const (
	WebsocketText   = websocket.TextMessage
	WebsocketBinary = websocket.BinaryMessage
)

// Do send a message to the websocket server and waits for the response. If the callback
// is not nil, then make sure you provide a context with deadline or timeout, otherwise
// you will leak goroutines.
func (wCtx *WebsocketCtx) Do(ctx context.Context, req WebsocketRequest) error {
	// run preflights
	for _, pre := range wCtx.cfg.preflights {
		pre(&req)
	}

	outC := wCtx.cfg.rpcOutFactory()
	if req.ID == "" {
		req.ID = utils.RandomDigit(10)
	}
	outC.InjectMessage(req.ReqMsg)
	outC.SetHdr(wCtx.cfg.predicateKey, req.Predicate)
	if tp := wCtx.cfg.tracePropagator; tp != nil {
		tp.Inject(ctx, containerTraceCarrier{out: outC})
	}
	for k, v := range req.ReqHdr {
		outC.SetHdr(k, v)
	}
	outC.SetID(req.ID)

	reqData, err := outC.Marshal()
	if err != nil {
		return err
	}

	wCtx.cMtx.Lock()
	wCtx.writeBytesTotal += uint64(len(reqData))
	wCtx.writeBytes += uint64(len(reqData))
	err = wCtx.c.WriteMessage(req.MessageType, reqData)
	wCtx.cMtx.Unlock()
	if err != nil {
		return err
	}

	outC.Release()

	if req.Callback != nil {
		go wCtx.waitForMessage(ctx, req.ID, req.ResMsg, req.Callback)
	}

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

var (
	ErrBadHandshake = websocket.ErrBadHandshake
	_               = ErrBadHandshake
)
