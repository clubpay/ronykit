package mw

import (
	"github.com/ronaksoft/ronykit"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func OpenTelemetry(tracerName string) func(svc ronykit.Service) ronykit.Service {
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

	return func(svc ronykit.Service) ronykit.Service {
		return serviceWrap{
			svc:  svc,
			pre:  pre,
			post: post,
		}
	}
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
