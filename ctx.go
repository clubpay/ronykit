package ronykit

import (
	"context"
	"math"
	"sync"

	"github.com/clubpay/ronykit/utils"
)

const (
	abortIndex = math.MaxInt >> 1
)

type (
	// ErrHandler is called when an error happens in internal layers.
	// NOTICE: ctx could be nil, make sure you do nil-check before calling its methods.
	ErrHandler       func(ctx *Context, err error)
	HandlerFunc      func(ctx *Context)
	HandlerFuncChain []HandlerFunc
)

type Context struct {
	utils.SpinLock
	ctx context.Context //nolint:containedctx
	cf  func()

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

func newContext() *Context {
	return &Context{
		kv:  make(map[string]interface{}),
		hdr: make(map[string]string),
	}
}

// execute the Context with the provided ExecuteArg. It implements ExecuteFunc
func (ctx *Context) execute(arg ExecuteArg) {
	ctx.wf = arg.WriteFunc
	ctx.handlers = append(ctx.handlers, arg.HandlerFuncChain...)
	ctx.Next()

	return
}

// Next sets the next handler which will be called after the current handler.
func (ctx *Context) Next() {
	ctx.index++
	for ctx.index <= len(ctx.handlers) {
		ctx.handlers[ctx.index-1](ctx)
		ctx.index++
	}
}

// StopExecution stops the execution of the next handlers, in other words, when you
// call this in your handler, any other middleware that are not executed will yet will
// be skipped over.
func (ctx *Context) StopExecution() {
	ctx.index = abortIndex
}

// AddModifier adds one or more modifiers to the context which will be executed on each outgoing
// Envelope before writing it to the wire.
func (ctx *Context) AddModifier(modifiers ...Modifier) {
	ctx.modifiers = append(ctx.modifiers, modifiers...)
}

func (ctx *Context) SetUserContext(userCtx context.Context) {
	ctx.ctx = userCtx
}

// Context returns a context.WithCancel which can be used a reference context for
// other context aware function calls. This context will be canceled at the end of
// Context lifetime.
func (ctx *Context) Context() context.Context {
	ctx.Lock()
	if ctx.ctx == nil {
		ctx.ctx, ctx.cf = context.WithCancel(context.Background())
	}
	ctx.Unlock()

	return ctx.ctx
}

// SetStatusCode set the connection status. It **ONLY** works if the underlying connection
// is a REST connection.
func (ctx *Context) SetStatusCode(code int) {
	rc, ok := ctx.Conn().(RESTConn)
	if !ok {
		return
	}

	rc.SetStatusCode(code)
}

// Conn returns the underlying connection
func (ctx *Context) Conn() Conn {
	return ctx.conn
}

// SetHdr sets the common header key-value pairs so in Out method we don't need to
// repeatedly set those. If you only want to set the header for an envelope, you can
// use Envelope.SetHdr method instead.
func (ctx *Context) SetHdr(k, v string) {
	ctx.hdr[k] = v
}

// SetHdrMap sets the common header key-value pairs so in Out method we don't need to
// repeatedly set those.
func (ctx *Context) SetHdrMap(hdr map[string]string) {
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
// This is useful for scenarios where you want to send cross-client message. For example,
// in a fictional chat server, you want to pass a message from client A to client B.
func (ctx *Context) OutTo(c Conn) *Envelope {
	return newEnvelope(ctx, c, true)
}

// Error is useful for some kind of errors which you are not going to return it to the connection,
// or you want to use its side effect for logging, monitoring etc. This will call your ErrHandler.
// The boolean result indicates if 'err' was an actual error.
func (ctx *Context) Error(err error) bool {
	if err != nil {
		ctx.err = err

		return true
	}

	return false
}

// HasError returns true if there is an error set by calling Error method.
func (ctx *Context) HasError() bool {
	return ctx.err != nil
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
	if ctx.cf != nil {
		ctx.cf()
	}
}

type ctxPool struct {
	sync.Pool
}

func (p *ctxPool) acquireCtx(c Conn) *Context {
	ctx, ok := p.Pool.Get().(*Context)
	if !ok {
		ctx = newContext()
	}

	ctx.conn = c
	ctx.in = newEnvelope(ctx, c, false)

	return ctx
}

func (p *ctxPool) releaseCtx(ctx *Context) {
	ctx.reset()
	p.Pool.Put(ctx)

	return
}
