package kit

import "context"

// LimitedContext is a wrapper around Context, which limit the capabilities of the original Context.
// This is useful in cases where we need to pass the Context, but we do not want to give access to all
// the exposed methods. For example, this is used in EdgeSelectorFunc.
type LimitedContext struct {
	ctx *Context
}

func newLimitedContext(ctx *Context) *LimitedContext {
	return &LimitedContext{
		ctx: ctx,
	}
}

// Context returns a context.WithCancel, which can be used a reference context for
// other context-aware function calls.
// This context will be canceled at the end of Context lifetime.
func (ctx *LimitedContext) Context() context.Context {
	return ctx.ctx.Context()
}

// In returns the incoming Envelope which contains the client's request.
func (ctx *LimitedContext) In() *Envelope {
	return ctx.ctx.In()
}

// Conn returns the underlying connection
func (ctx *LimitedContext) Conn() Conn {
	return ctx.ctx.conn
}

// SetHdr sets the common header key-value pairs, so in Out method we do not need to
// repeatedly set those. If you only want to set the header for an envelope, you can
// use Envelope.SetHdr method instead.
func (ctx *LimitedContext) SetHdr(k, v string) {
	ctx.ctx.hdr[k] = v
}

// SetHdrMap sets the common header key-value pairs, so in Out method we do not need to
// repeatedly set those.
func (ctx *LimitedContext) SetHdrMap(hdr map[string]string) {
	for k, v := range hdr {
		ctx.ctx.hdr[k] = v
	}
}

func (ctx *LimitedContext) Route() string {
	return ctx.ctx.Route()
}

func (ctx *LimitedContext) ServiceName() string {
	return ctx.ctx.ServiceName()
}

func (ctx *LimitedContext) ClusterID() string {
	return ctx.ctx.ClusterID()
}

func (ctx *LimitedContext) ClusterMembers() ([]string, error) {
	return ctx.ctx.ClusterMembers()
}

func (ctx *LimitedContext) ClusterStore() ClusterStore {
	return ctx.ctx.ClusterStore()
}

func (ctx *LimitedContext) SetUserContext(userCtx context.Context) {
	ctx.ctx.SetUserContext(userCtx)
}
