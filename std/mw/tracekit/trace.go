package tracekit

import (
	"strings"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/mw"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	traceparentHeader = "traceparent"
	tracestateHeader  = "tracestate"
)

func Trace(tracerName string) func(svc ronykit.Service) ronykit.Service {
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
		return mw.Wrap(svc, pre, post)
	}
}

type ctxCarrier struct {
	traceParent string
	traceState  string
	ctx         *ronykit.Context
}

func newCtxCarrier(ctx *ronykit.Context) ctxCarrier {
	c := ctxCarrier{ctx: ctx}
	c.ctx.Conn().Walk(
		func(key string, v string) bool {
			if strings.EqualFold(traceparentHeader, key) {
				c.traceParent = v
			} else if strings.EqualFold(tracestateHeader, key) {
				c.traceState = v
			}

			return true
		},
	)

	return c
}

func (c ctxCarrier) Get(key string) string {
	switch key {
	case traceparentHeader:
		return c.traceParent
	case tracestateHeader:
		return c.traceState
	}

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
	c.ctx.Conn().Walk(
		func(key string, _ string) bool {
			keys = append(keys, key)

			return true
		},
	)

	return keys
}
