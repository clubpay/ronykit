package ronykit

import (
	"sync"
	"sync/atomic"

	log "github.com/ronaksoft/golog"
	"go.uber.org/zap"
)

type northBridge struct {
	ctxPool sync.Pool
	l       log.Logger
	b       Bundle
	eh      ErrHandler
	opened  int64
	closed  int64
}

func (n *northBridge) OnOpen(c Conn) {
	atomic.AddInt64(&n.opened, 1)
}

func (n *northBridge) OnClose(connID uint64) {
	atomic.AddInt64(&n.closed, 1)
}

func (n *northBridge) OnMessage(c Conn, msg []byte) error {
	if ce := n.l.Check(log.DebugLevel, "received message"); ce != nil {
		ce.Write(
			zap.Uint64("ConnID", c.ConnID()),
		)
	}

	dispatchFunc, err := n.b.Dispatch(c, msg)
	if err != nil {
		return err
	}

	ctx := n.acquireCtx(c)
	err = dispatchFunc(
		ctx,
		func(m Message, writeFunc WriteFunc, handlers ...Handler) {
			ctx.in = m
			ctx.wf = writeFunc
		Loop:
			for idx := range handlers {
				h := handlers[idx]
				for h != nil {
					h = h(ctx)
					if ctx.stopped {
						break Loop
					}
				}
			}

			return
		},
	)
	n.releaseCtx(ctx)

	return err
}

func (n *northBridge) acquireCtx(c Conn) *Context {
	ctx, ok := n.ctxPool.Get().(*Context)
	if !ok {
		ctx = &Context{
			kv: make(map[string]interface{}),
		}
	}

	ctx.conn = c

	return ctx
}

func (n *northBridge) releaseCtx(ctx *Context) {
	ctx.reset()
	n.ctxPool.Put(ctx)

	return
}
