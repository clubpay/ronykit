package mw

import (
	"github.com/ronaksoft/ronykit"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func OpenTelemetry(tracerName string) func(srv ronykit.IService) ronykit.IService {
	traceCtx := propagation.TraceContext{}
	tracer := otel.GetTracerProvider().Tracer(tracerName)
	pre := func(ctx *ronykit.Context) ronykit.Handler {
		userCtx := traceCtx.Extract(ctx.Context(), newCtxCarrier(ctx))
		userCtx, _ = tracer.Start(userCtx, ctx.Route())
		ctx.SetUserContext(userCtx)

		return nil
	}
	post := func(ctx *ronykit.Context) ronykit.Handler {
		span := trace.SpanFromContext(ctx.Context())
		span.End()

		return nil
	}

	return func(srv ronykit.IService) ronykit.IService {
		return serviceWrap{
			srv:  srv,
			pre:  pre,
			post: post,
		}
	}
}

type serviceWrap struct {
	srv  ronykit.IService
	pre  ronykit.Handler
	post ronykit.Handler
}

func (s serviceWrap) Name() string {
	return s.srv.Name()
}

func (s serviceWrap) Routes() []ronykit.IRoute {
	return s.srv.Routes()
}

func (s serviceWrap) PreHandlers() []ronykit.Handler {
	var handlers = []ronykit.Handler{s.pre}

	return append(handlers, s.srv.PreHandlers()...)
}

func (s serviceWrap) PostHandlers() []ronykit.Handler {
	var handlers = []ronykit.Handler{s.post}

	return append(handlers, s.srv.PostHandlers()...)
}

type ctxCarrier struct {
	ctx *ronykit.Context
}

func newCtxCarrier(ctx *ronykit.Context) ctxCarrier {
	return ctxCarrier{ctx: ctx}
}

func (c ctxCarrier) Get(key string) string {
	v, ok := c.ctx.Get(key).(string)
	if !ok {
		return ""
	}

	return v
}

func (c ctxCarrier) Set(key string, value string) {
	c.ctx.Set(key, value)
}

func (c ctxCarrier) Keys() []string {
	var keys []string
	c.ctx.Walk(
		func(key string, _ interface{}) bool {
			keys = append(keys, key)

			return true
		},
	)

	return keys
}
