package tracekit

import (
	"github.com/clubpay/ronykit/kit"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func Span(ctx *kit.Context) trace.Span {
	return trace.SpanFromContext(ctx.Context())
}

func Link(ctx *kit.Context, kv ...attribute.KeyValue) trace.Link {
	return trace.LinkFromContext(ctx.Context(), kv...)
}
