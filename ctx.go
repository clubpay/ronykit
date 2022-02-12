package ronykit

import (
	"context"

	"github.com/ronaksoft/ronykit/utils"
)

type Context struct {
	utils.SpinLock

	nb      *northBridge
	kv      map[string]interface{}
	hdr     map[string]string
	conn    Conn
	in      *Envelope
	wf      WriteFunc
	err     error
	stopped bool
	next    Handler
	ctx     context.Context
}

// Context returns a context.Background which can be used a reference context for
// other context aware function calls. You can also replace it with your own context
// using SetUserContext function.
func (ctx *Context) Context() context.Context {
	return ctx.ctx
}

// Next sets the next handler which will be called after the current handler.
func (ctx *Context) Next(h Handler) {
	ctx.next = h
}

// StopExecution stops the execution of the next handlers.
func (ctx *Context) StopExecution() {
	ctx.stopped = true
}

// SetUserContext replaces the default context with the provided context.
func (ctx *Context) SetUserContext(userCtx context.Context) {
	ctx.ctx = userCtx
}

// SetStatusCode set the connection status. It **ONLY** works if the underlying connection
// is a REST connection.
func (ctx *Context) SetStatusCode(code int) {
	rc, ok := ctx.Conn().(RESTConn)
	if ok {
		rc.SetStatusCode(code)
	}
}

// Conn returns the underlying connection
func (ctx *Context) Conn() Conn {
	return ctx.conn
}

// SetHdr sets the common header key-value pairs so in Out method we don't need to
// repeatedly set those.
func (ctx *Context) SetHdr(hdr map[string]string) {
	for k, v := range hdr {
		ctx.hdr[k] = v
	}
}

// In returns the incoming Envelope which received from the connection.
// You **SHOULD NOT** use this envelope to write data to the connection. If you want
// to return a message/envelope to connection use Out or OutTo methods.
func (ctx *Context) In() *Envelope {
	return ctx.in
}

// Out generate a new Envelope which could be used to send data to the connection.
func (ctx *Context) Out() *Envelope {
	return ctx.OutTo(ctx.conn)
}

// OutTo is similar to Out except that it lets you send your envelope to other connection.
func (ctx *Context) OutTo(c Conn) *Envelope {
	return newEnvelope(ctx, c, true)
}

// Error is useful for some kind of errors which you are not going to return it to the connection,
// or you want to use its side effect for logging, monitoring etc. This will call your ErrHandler.
// The boolean result indicates if 'err' was an actual error.
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

func (ctx *Context) reset() {
	for k := range ctx.kv {
		delete(ctx.kv, k)
	}
	for k := range ctx.hdr {
		delete(ctx.hdr, k)
	}

	ctx.in.release()
	ctx.next = nil
	ctx.stopped = false
	ctx.ctx = nil
}
