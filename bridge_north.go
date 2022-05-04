package ronykit

import (
	"runtime/debug"
	"sync"
)

type northBridge struct {
	ctxPool sync.Pool
	l       Logger
	b       Gateway
	eh      ErrHandler
}

var _ GatewayDelegate = (*northBridge)(nil)

func (n *northBridge) OnOpen(c Conn) {
	// Maybe later we can do something
}

func (n *northBridge) OnClose(connID uint64) {
	// Maybe later we can do something
}

func (n *northBridge) recoverPanic(ctx *Context, c Conn) {
	if r := recover(); r != nil {
		n.l.Errorf("Panic Recovered: [%s](%s) : %s\nError: ", ctx.ServiceName(), ctx.Route(), c.ClientIP(), r)
		n.l.Errorf("Stack Trace: \n%s", debug.Stack())
	}
}

func (n *northBridge) OnMessage(c Conn, msg []byte) {
	ctx := n.acquireCtx(c)
	defer n.recoverPanic(ctx, c)

	dispatchFunc, err := n.b.Dispatch(c, msg)
	if err != nil {
		n.eh(ctx, err)
		n.releaseCtx(ctx)

		return
	}

	err = dispatchFunc(ctx, n.execFunc)
	if err != nil {
		n.eh(ctx, err)
	}

	n.releaseCtx(ctx)

	return
}

func (n *northBridge) execFunc(ctx *Context, writeFunc WriteFunc, handlers ...HandlerFunc) {
	ctx.wf = writeFunc
	ctx.handlers = append(ctx.handlers, handlers...)
	ctx.Next()

	return
}

func (n *northBridge) acquireCtx(c Conn) *Context {
	ctx, ok := n.ctxPool.Get().(*Context)
	if !ok {
		ctx = newContext()
	}

	ctx.conn = c
	ctx.in = newEnvelope(ctx, c, false)

	return ctx
}

func (n *northBridge) releaseCtx(ctx *Context) {
	ctx.reset()
	n.ctxPool.Put(ctx)

	return
}
