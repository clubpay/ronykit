package edge

import (
	"github.com/ronaksoft/ronykit"
	"sync"
)

type northBridge struct {
	ctxPool sync.Pool
	r       Router
	ep      ronykit.EnvelopePool
	gw      ronykit.Gateway
	d       ronykit.Dispatcher
}

func (n *northBridge) OnOpen(c ronykit.Conn) {
	n.d.OnOpen(c)
}

func (n *northBridge) OnClose(connID uint64) {
	// TODO:: do we need to do anything ?
}

func (n *northBridge) OnMessage(c ronykit.Conn, streamID int64, msg []byte) error {
	ctx := n.acquireCtx(c, streamID)
	err := n.d.Deserialize(
		c, msg,
		func(envelope ronykit.Envelope) error {
			return n.execute(ctx, envelope)
		},
	)
	n.releaseCtx(ctx)

	return err
}

func (n *northBridge) execute(ctx *RequestCtx, envelope ronykit.Envelope) error {
	handlers, err := n.r.Route(envelope)
	if err != nil {
		return err
	}

Loop:
	for idx := range handlers {
		h := handlers[idx]
		for h != nil {
			h = h(ctx, envelope)
			if ctx.stopped {
				break Loop
			}
		}
	}

	return ctx.Flush()
}

func (n *northBridge) acquireCtx(c ronykit.Conn, streamID int64) *RequestCtx {
	ctx, ok := n.ctxPool.Get().(*RequestCtx)
	if !ok {
		ctx = &RequestCtx{}
	}

	ctx.streamID = streamID
	ctx.conn = c

	return ctx
}

func (n *northBridge) releaseCtx(ctx *RequestCtx) {
	if n.ep != nil {
		for _, e := range ctx.outBuf {
			n.ep.Release(e)
		}
	}

	ctx.outBuf = ctx.outBuf[:0]
	ctx.conn = nil
	ctx.stopped = false

	n.ctxPool.Put(ctx)
}
