package tracekit

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestSpanHelpers(t *testing.T) {
	SetInstrument("test")
	otelCtx := context.Background()
	ctx, span, end := NewSpan(otelCtx, "op")
	if span == nil {
		t.Fatalf("expected span")
	}
	end()

	Span(ctx, String("k1", "v1"))
	Error(span, errors.New("boom"))
}

func TestLink(t *testing.T) {
	ctx := trace.ContextWithSpan(context.Background(), trace.SpanFromContext(context.Background()))
	link := Link(ctx, String("k1", "v1"))
	if !link.SpanContext.IsValid() && len(link.Attributes) == 0 {
		t.Fatalf("expected link to carry attributes or span context")
	}
}
