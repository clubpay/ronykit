package ronykit

import (
	"github.com/ronaksoft/ronykit/utils"
)

type Context struct {
	utils.SpinLock

	nb      *northBridge
	kv      map[string]interface{}
	conn    Conn
	in      Message
	wf      WriteFunc
	eh      ErrHandler
	stopped bool
}

func (ctx *Context) Set(key string, val interface{}) {
	ctx.Lock()
	ctx.kv[key] = val
	ctx.Unlock()
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

func (ctx *Context) Receive() Message {
	return ctx.in
}

func (ctx *Context) Send(m Message, ctxKeys ...string) {
	ctx.Error(ctx.wf(m, ctxKeys...))
}

func (ctx *Context) Error(err error) bool {
	if err != nil {
		if h := ctx.nb.eh; h != nil {
			h(err)
		}

		return true
	}

	return false
}

// StopExecution stops the execution of the next handlers.
func (ctx *Context) StopExecution() {
	ctx.stopped = true
}
