package ronykit

import (
	"context"

	"github.com/ronaksoft/ronykit/utils"
)

type Context struct {
	utils.SpinLock

	nb      *northBridge
	kv      map[string]interface{}
	conn    Conn
	in      *Envelope
	wf      WriteFunc
	err     error
	stopped bool
	next    Handler
	ctx     context.Context
}

func (ctx *Context) Context() context.Context {
	return ctx.ctx
}

func (ctx *Context) SetUserContext(userCtx context.Context) {
	ctx.ctx = userCtx
}

func (ctx *Context) Next(h Handler) {
	ctx.next = h
}

func (ctx *Context) SetStatusCode(code int) {
	rc, ok := ctx.Conn().(REST)
	if ok {
		rc.SetStatusCode(code)
	}
}

func (ctx *Context) Conn() Conn {
	return ctx.conn
}

func (ctx *Context) In() *Envelope {
	return ctx.in
}

func (ctx *Context) Out() *Envelope {
	return newEnvelope(ctx, ctx.conn)
}

func (ctx *Context) OutTo(c Conn) *Envelope {
	return newEnvelope(ctx, c)
}

func (ctx *Context) Error(err error) bool {
	if err != nil {
		ctx.err = err
		if h := ctx.nb.eh; h != nil {
			h(ctx, err)
		}

		return true
	}

	return false
}

// StopExecution stops the execution of the next handlers.
func (ctx *Context) StopExecution() {
	ctx.stopped = true
}

func (ctx *Context) reset() {
	for k := range ctx.kv {
		delete(ctx.kv, k)
	}

	ctx.in.Release()
	ctx.next = nil
	ctx.stopped = false
	ctx.ctx = nil
}
