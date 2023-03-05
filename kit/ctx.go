package kit

import (
	"context"
	"math"
	"net/http"
	"sync"

	"github.com/clubpay/ronykit/kit/utils"
)

const (
	abortIndex = math.MaxInt >> 1
)

type (
	// ErrHandlerFunc is called when an error happens in internal layers.
	// NOTICE: ctx could be nil, make sure you do nil-check before calling its methods.
	ErrHandlerFunc func(ctx *Context, err error)
	// HandlerFunc is a function that will execute code in its context. If there is another handler
	// set in the path, by calling ctx.Next you can move forward and then run the rest of the code in
	// your handler.
	HandlerFunc        func(ctx *Context)
	HandlerFuncChain   []HandlerFunc
	LimitedHandlerFunc func(ctx *LimitedContext)
)

type Context struct {
	utils.SpinLock
	ctx       context.Context //nolint:containedctx
	cf        func()
	sb        *southBridge
	ls        *localStore
	forwarded bool

	serviceName []byte
	contractID  []byte
	route       []byte
	rawData     []byte

	kv         map[string]interface{}
	hdr        map[string]string
	conn       Conn
	in         *Envelope
	wf         WriteFunc
	modifiers  []ModifierFunc
	err        error
	statusCode int

	handlers     HandlerFuncChain
	handlerIndex int
}

func newContext(ls *localStore) *Context {
	return &Context{
		ls:         ls,
		kv:         make(map[string]interface{}, 4),
		hdr:        make(map[string]string, 4),
		statusCode: http.StatusOK,
	}
}

type ExecuteArg struct {
	ServiceName string
	ContractID  string
	Route       string
}

// execute the Context with the provided ExecuteArg. It implements ExecuteFunc
func (ctx *Context) execute(arg ExecuteArg, c Contract) {
	ctx.
		setRoute(arg.Route).
		setServiceName(arg.ServiceName).
		setContractID(arg.ContractID).
		AddModifier(c.Modifiers()...)

	ctx.handlers = append(ctx.handlers, c.Handlers()...)
	ctx.Next()
}

type executeRemoteArg struct {
	Target      string
	In          *envelopeCarrier
	OutCallback func(carrier *envelopeCarrier)
}

func (ctx *Context) executeRemote(arg executeRemoteArg) error {
	if ctx.sb == nil {
		return ErrSouthBridgeDisabled
	}

	ch, err := ctx.sb.sendMessage(
		arg.In.SessionID,
		arg.Target,
		arg.In.ToJSON(),
	)
	if err != nil {
		return err
	}

	// FixME: this can block forever
	for c := range ch {
		arg.OutCallback(c)
	}

	return nil
}

// Next sets the next handler which will be called after the current handler.
func (ctx *Context) Next() {
	ctx.handlerIndex++
	for ctx.handlerIndex <= len(ctx.handlers) {
		ctx.handlers[ctx.handlerIndex-1](ctx)
		ctx.handlerIndex++
	}
}

// StopExecution stops the execution of the next handlers, in other words, when you
// call this in your handler, any other middleware that are not executed will yet will
// be skipped over.
func (ctx *Context) StopExecution() {
	ctx.handlerIndex = abortIndex
}

// AddModifier adds one or more modifiers to the context which will be executed on each outgoing
// Envelope before writing it to the wire.
func (ctx *Context) AddModifier(modifiers ...ModifierFunc) {
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
	ctx.statusCode = code

	rc, ok := ctx.Conn().(RESTConn)
	if !ok {
		return
	}

	rc.SetStatusCode(code)
}

func (ctx *Context) GetStatusCode() int {
	return ctx.statusCode
}

func (ctx *Context) GetStatusText() string {
	return http.StatusText(ctx.statusCode)
}

// Conn returns the underlying connection
func (ctx *Context) Conn() Conn {
	return ctx.conn
}

// RESTConn returns the underlying REST connection. It panics if the underlying connection
// does not implement RESTConn interface.
func (ctx *Context) RESTConn() RESTConn {
	return ctx.conn.(RESTConn) //nolint:forcetypeassert
}

func (ctx *Context) IsREST() bool {
	_, ok := ctx.conn.(RESTConn)

	return ok
}

// PresetHdr sets the common header key-value pairs so in Out method we don't need to
// repeatedly set those. This method is useful for some cases if we need to update the
// header in some middleware before the actual response is prepared.
// If you only want to set the header for an envelope, you can use Envelope.SetHdr method instead.
func (ctx *Context) PresetHdr(k, v string) {
	ctx.hdr[k] = v
}

// PresetHdrMap sets the common header key-value pairs so in Out method we don't need to
// repeatedly set those. Please refer to PresetHdr for more details
func (ctx *Context) PresetHdrMap(hdr map[string]string) {
	for k, v := range hdr {
		ctx.hdr[k] = v
	}
}

// In returns the incoming Envelope which received from the connection.
// You **MUST NOT** call Send method of this Envelope.
// If you want to return a message/envelope to connection use Out or OutTo methods
// of the Context
func (ctx *Context) In() *Envelope {
	return ctx.in
}

// InputRawData returns the raw input data from the connection. This slice is not valid
// after this Context lifetime. If you need to use it after the Context lifetime,
// you need to copy it.
// You should not use this method in your code, ONLY if you need it for debugging.
func (ctx *Context) InputRawData() []byte {
	return ctx.rawData
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
// or you want to use its side effect for logging, monitoring etc. This will call your ErrHandlerFunc.
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

// Limited returns a LimitedContext. This is useful when you don't want to give all
// capabilities of the Context to some other function/method.
func (ctx *Context) Limited() *LimitedContext {
	return newLimitedContext(ctx)
}

func (ctx *Context) reset() {
	for k := range ctx.kv {
		delete(ctx.kv, k)
	}
	for k := range ctx.hdr {
		delete(ctx.hdr, k)
	}

	ctx.forwarded = false
	ctx.serviceName = ctx.serviceName[:0]
	ctx.contractID = ctx.contractID[:0]
	ctx.route = ctx.route[:0]

	ctx.in.release()
	ctx.statusCode = http.StatusOK
	ctx.handlerIndex = 0
	ctx.handlers = ctx.handlers[:0]
	ctx.modifiers = ctx.modifiers[:0]
	if ctx.cf != nil {
		ctx.cf()
		ctx.cf = nil
	}
	ctx.ctx = nil
}

func (ctx *Context) isREST() bool {
	_, ok := ctx.Conn().(RESTConn)

	return ok
}

type ctxPool struct {
	sync.Pool
	ls *localStore
	th HandlerFunc // trace handler
}

func (p *ctxPool) acquireCtx(c Conn) *Context {
	ctx, ok := p.Pool.Get().(*Context)
	if !ok {
		ctx = newContext(p.ls)
	}

	ctx.conn = c
	ctx.in = newEnvelope(ctx, c, false)
	if p.th != nil {
		ctx.handlers = append(ctx.handlers, p.th)
	}

	return ctx
}

func (p *ctxPool) releaseCtx(ctx *Context) {
	ctx.reset()
	p.Pool.Put(ctx)
}
