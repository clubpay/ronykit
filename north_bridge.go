package ronykit

import (
	"context"
	"runtime/debug"
	"sync"

	log "github.com/ronaksoft/golog"
	"go.uber.org/zap"
)

type northBridge struct {
	ctxPool sync.Pool
	l       log.Logger
	b       Bundle
	eh      ErrHandler
}

func (n *northBridge) OnOpen(c Conn) {
	// Maybe later we can do something
}

func (n *northBridge) OnClose(connID uint64) {
	// Maybe later we can do something
}

func (n *northBridge) recoverPanic(ctx *Context, c Conn) {
	if r := recover(); r != nil {
		n.l.Error("Panic Recovered",
			zap.String("ClientIP", c.ClientIP()),
			zap.String("Route", ctx.Route()),
			zap.String("Service", ctx.ServiceName()),
			zap.Uint64("ConnID", c.ConnID()),
			zap.Any("Error", r),
		)

		n.l.Error("Panic Stack Trace", zap.ByteString("Stack", debug.Stack()))
	}
}

func (n *northBridge) OnMessage(c Conn, msg []byte) {
	ctx := n.acquireCtx(c)
	defer n.recoverPanic(ctx, c)

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

					h, ctx.next = ctx.next, nil
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
			kv:  make(map[string]interface{}),
			hdr: make(map[string]string),
		}
	}

	ctx.conn = c
	ctx.in = newEnvelope(ctx, c)
	ctx.ctx = context.Background()

	return ctx
}

func (n *northBridge) releaseCtx(ctx *Context) {
	ctx.reset()
	n.ctxPool.Put(ctx)

	return
}
