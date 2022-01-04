package tracekit

import (
	"github.com/ronaksoft/ronykit"
	"go.opentelemetry.io/otel/trace"
)

func Span(ctx *ronykit.Context) trace.Span {
	return trace.SpanFromContext(ctx.Context())
}
