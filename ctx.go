package ronykit

import (
	"context"

	"github.com/ronaksoft/ronykit/utils"
)

const (
	CtxServiceName = "__ServiceName__"
	CtxRoute       = "__Route__"
)

type Context struct {
	utils.SpinLock

	nb      *northBridge
	kv      map[string]interface{}
	conn    Conn
	in      Message
	wf      WriteFunc
	err     error
	stopped bool
	ctx     context.Context
}

func (ctx *Context) Context() context.Context {
	return ctx.ctx
}

func (ctx *Context) SetUserContext(userCtx context.Context) {
	ctx.ctx = userCtx
}

func (ctx *Context) Route() string {
	return ctx.Get(CtxRoute).(string)
}

func (ctx *Context) ServiceName() string {
	return ctx.Get(CtxServiceName).(string)
}

func (ctx *Context) Set(key string, val interface{}) {
	ctx.Lock()
	ctx.kv[key] = val
	ctx.Unlock()
}

func (ctx *Context) SetStatusCode(code int) {
	rc, ok := ctx.Conn().(REST)
	if ok {
		rc.SetStatusCode(code)
	}
}

func (ctx *Context) Get(key string) interface{} {
	ctx.Lock()
	v := ctx.kv[key]
	ctx.Unlock()

	return v
}

func (ctx *Context) Walk(f func(key string, val interface{}) bool) {
	for k, v := range ctx.kv {
		if !f(k, v) {
			return
		}
	}
}

func (ctx *Context) Conn() Conn {
	return ctx.conn
}

func (ctx *Context) Receive() Message {
	return ctx.in
}

func (ctx *Context) Send(m Message, ctxKeys ...string) {
	ctx.Error(ctx.wf(m, ctxKeys...))
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

	ctx.stopped = false
	ctx.ctx = nil
}
