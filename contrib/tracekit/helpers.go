package tracekit

import (
	"github.com/clubpay/ronykit/kit"
	"go.opentelemetry.io/otel/trace"
)

func Span(ctx *kit.Context) trace.Span {
	return trace.SpanFromContext(ctx.Context())
}
