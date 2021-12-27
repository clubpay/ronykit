package ronykit

import (
	"sync"

	"github.com/ronaksoft/ronykit/utils"
)

type Context struct {
	utils.SpinLock

	streamID  int64
	conn      Conn
	err       error
	nb        *northBridge
	kv        map[string]interface{}
	in        Message
	writeFunc WriteFunc
	stopped   bool
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

func (ctx *Context) Send(m Message) {
	ctx.writeFunc(m)
}

func (ctx *Context) Error(err error) {
	if err != nil {
		ctx.err = err
	}
}

// StopExecution stops the execution of the next handlers.
func (ctx *Context) StopExecution() {
	ctx.stopped = true
}

var ctxPool sync.Pool

func acquireCtx(c Conn, streamID int64) *Context {
	ctx, ok := ctxPool.Get().(*Context)
	if !ok {
		ctx = &Context{
			kv: make(map[string]interface{}),
		}
	}

	ctx.conn = c
	ctx.streamID = streamID

	return ctx
}

func releaseCtx(ctx *Context) {
	for k := range ctx.kv {
		delete(ctx.kv, k)
	}
	ctx.streamID = 0
	ctx.stopped = false
	ctx.err = nil
	ctxPool.Put(ctx)

	return
}
