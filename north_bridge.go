package ronykit

import (
	"context"
	"sync"
	"sync/atomic"

	log "github.com/ronaksoft/golog"
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

func (n *northBridge) OnMessage(c Conn, msg []byte) {
	ctx := n.acquireCtx(c)
	dispatchFunc, err := n.b.Dispatch(c, msg)
	if err != nil {
		if n.eh != nil {
			n.eh(ctx, err)
		}

		n.releaseCtx(ctx)

		return
	}

	err = dispatchFunc(
		ctx,
		func(writeFunc WriteFunc, handlers ...Handler) {
			ctx.wf = writeFunc
		Loop:
			for idx := range handlers {
				h := handlers[idx]
				for h != nil {
					h(ctx)
					if ctx.stopped {
						break Loop
					}

					if ctx.next == nil {
						break
					}

					h = ctx.next
					ctx.next = nil
				}
			}

			return
		},
	)
	if err != nil {
		if n.eh != nil {
			n.eh(ctx, err)
		}
	}

	n.releaseCtx(ctx)

	return
}

func (n *northBridge) acquireCtx(c Conn) *Context {
	ctx, ok := n.ctxPool.Get().(*Context)
	if !ok {
		ctx = &Context{
			kv: make(map[string]interface{}),
		}
	}

	ctx.conn = c
	ctx.in = newEnvelope(ctx)
	ctx.ctx = context.Background()

	return ctx
}

func (n *northBridge) releaseCtx(ctx *Context) {
	ctx.reset()
	n.ctxPool.Put(ctx)

	return
}
