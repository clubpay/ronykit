package ronykit

import (
	"context"
	"math"

	"github.com/clubpay/ronykit/utils"
)

type (
	ErrHandler       func(ctx *Context, err error)
	HandlerFunc      func(ctx *Context)
	HandlerFuncChain []HandlerFunc
)

type Context struct {
	utils.SpinLock
	ctx context.Context //nolint:containedctx

	nb        *northBridge
	kv        map[string]interface{}
	hdr       map[string]string
	conn      Conn
	in        *Envelope
	wf        WriteFunc
	modifiers []Modifier
	err       error

	handlers HandlerFuncChain
	index    int
}

const (
	abortIndex = math.MaxInt >> 1
)

// Context returns a context.Background which can be used a reference context for
// other context aware function calls. You can also replace it with your own context
// using SetUserContext function.
func (ctx *Context) Context() context.Context {
	return ctx.ctx
}

// Next sets the next handler which will be called after the current handler.
func (ctx *Context) Next() {
	ctx.index++
	for ctx.index <= len(ctx.handlers) {
		ctx.handlers[ctx.index-1](ctx)
		ctx.index++
	}
}

// StopExecution stops the execution of the next handlers.
func (ctx *Context) StopExecution() {
	ctx.index = abortIndex
}

// AddModifier adds one or more modifiers to the context which will be executed on each outgoing
// Envelope before writing it to the wire.
func (ctx *Context) AddModifier(modifiers ...Modifier) {
	ctx.modifiers = append(ctx.modifiers, modifiers...)
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

// SetHdrMap sets the common header key-value pairs so in Out method we don't need to
// repeatedly set those.
func (ctx *Context) SetHdrMap(hdr map[string]string) {
	for k, v := range hdr {
		ctx.hdr[k] = v
	}
}

// SetHdr sets the common header key-value pairs so in Out method we don't need to
// repeatedly set those.
func (ctx *Context) SetHdr(k, v string) {
	ctx.hdr[k] = v
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
	ctx.index = 0
	ctx.handlers = ctx.handlers[:0]
	ctx.modifiers = ctx.modifiers[:0]
	ctx.ctx = nil
}
