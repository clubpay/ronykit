package stub

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/utils"
	"github.com/clubpay/ronykit/utils/reflector"
	"github.com/fasthttp/websocket"
	"github.com/goccy/go-json"
)

type WebsocketCtx struct {
	cfg     wsConfig
	err     *Error
	r       *reflector.Reflector
	dumpReq io.Writer
	dumpRes io.Writer

	pendingL sync.Mutex
	pending  map[string]chan ronykit.IncomingRPCContainer

	// fasthttp entities
	url string
	c   *websocket.Conn
}

func (wCtx *WebsocketCtx) Connect(ctx context.Context, path string) error {
	path = strings.TrimLeft(path, "/")
	url := wCtx.url
	if path != "" {
		url = fmt.Sprintf("%s/%s", wCtx.url, path)
	}
	c, _, err := wCtx.cfg.d.DialContext(ctx, url, wCtx.cfg.upgradeHdr)
	if err != nil {
		return err
	}

	wCtx.c = c

	// run receiver in background
	go wCtx.receiver()

	return nil
}

func (wCtx *WebsocketCtx) Disconnect() error {
	return wCtx.c.Close()
}

func (wCtx *WebsocketCtx) receiver() {
	for {
		_, p, err := wCtx.c.ReadMessage()
		if err != nil || len(p) == 0 {
			// TODO:: maybe error handler ?
			return
		}

		c := wCtx.cfg.rpcInFactory()
		err = c.Unmarshal(p)
		if err != nil {
			// TODO:: maybe error handler ?
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
	ctx context.Context, predicate string, req, res ronykit.Message,
	cb RPCMessageHandler,
) error {
	return wCtx.do(ctx, predicate, websocket.TextMessage, req, res, cb)
}

func (wCtx *WebsocketCtx) BinaryMessage(
	ctx context.Context, predicate string, req, res ronykit.Message,
	cb RPCMessageHandler,
) error {
	return wCtx.do(ctx, predicate, websocket.BinaryMessage, req, res, cb)
}

func (wCtx *WebsocketCtx) do(
	ctx context.Context, predicate string, msgType int, req, res ronykit.Message,
	cb RPCMessageHandler,
) error {
	outC := wCtx.cfg.rpcOutFactory()

	payloadData, err := ronykit.MarshalMessage(req)
	if err != nil {
		return err
	}
	id := utils.RandomDigit(10)
	outC.SetPayload(json.RawMessage(payloadData))
	outC.SetHdr(wCtx.cfg.predicateKey, predicate)
	outC.SetID(id)

	reqData, err := outC.Marshal()
	if err != nil {
		return err
	}

	err = wCtx.c.WriteMessage(msgType, reqData)
	if err != nil {
		return err
	}

	outC.Release()

	go wCtx.waitForMessage(ctx, id, res, cb)

	return nil
}

func (wCtx *WebsocketCtx) waitForMessage(
	ctx context.Context, id string, res ronykit.Message, cb RPCMessageHandler,
) {
	resCh := make(chan ronykit.IncomingRPCContainer, 1)
	wCtx.pendingL.Lock()
	wCtx.pending[id] = resCh
	wCtx.pendingL.Unlock()

	select {
	case c := <-resCh:
		err := c.Fill(res)
		cb(ctx, res, c.GetHdrMap(), err)

	case <-ctx.Done():
	}

	wCtx.pendingL.Lock()
	delete(wCtx.pending, id)
	wCtx.pendingL.Unlock()
}
