package tracekit

import (
	"context"

	"github.com/clubpay/ronykit/kit"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.11.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	w3cTraceParent = "traceparent"
	w3cState       = "tracestate"
	b3Single       = "b3"
	b3TraceID      = "x-b3-traceid"
	b3ParentSpanID = "x-b3-parentspanid"
	b3SpanID       = "x-b3-spanid"
	b3Sampled      = "x-b3-sampled"
	b3Flags        = "x-b3-flags"
)

type TracePropagator int

const (
	w3cPropagator TracePropagator = iota
	b3Propagator
)

func B3(name string, opts ...Option) kit.Tracer {
	cfg := &config{
		tracerName: name,
		propagator: b3Propagator,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return withTracer(cfg)
}

func W3C(name string, opts ...Option) kit.Tracer {
	cfg := &config{
		tracerName: name,
		propagator: w3cPropagator,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return withTracer(cfg)
}

func withTracer(cfg *config) kit.Tracer {
	t := tracer{
		name:    cfg.tracerName,
		dynTags: cfg.dynTags,
	}
	for k, v := range cfg.tags {
		t.kvs = append(t.kvs, attribute.String(k, v))
	}

	if cfg.serviceName != "" {
		t.kvs = append(t.kvs, semconv.ServiceNameKey.String(cfg.serviceName))
	}
	if cfg.env != "" {
		t.kvs = append(t.kvs, semconv.DeploymentEnvironmentKey.String(cfg.env))
	}

	if len(t.kvs) > 0 {
		t.spanOpts = append(t.spanOpts, trace.WithAttributes(t.kvs...))
	}

	switch cfg.propagator {
	case b3Propagator:
		t.p = b3.New(b3.WithInjectEncoding(b3.B3SingleHeader))
	default:
		t.p = propagation.TraceContext{}
	}

	return &t
}

type tracer struct {
	name     string
	p        propagation.TextMapPropagator
	spanOpts []trace.SpanStartOption
	kvs      []attribute.KeyValue
	dynTags  func(ctx *kit.LimitedContext) map[string]string
}

func (t *tracer) Handler() kit.HandlerFunc {
	return func(ctx *kit.Context) {
		userCtx, span := otel.Tracer(t.name).
			Start(
				t.p.Extract(ctx.Context(), connCarrier{ctx.Conn()}),
				ctx.Route(),
				t.spanOpts...,
			)

		if t.dynTags != nil {
			dynTags := t.dynTags(ctx.Limited())
			kvs := make([]attribute.KeyValue, 0, len(dynTags))
			for k, v := range dynTags {
				kvs = append(kvs, attribute.String(k, v))
			}
			span.SetAttributes(kvs...)
		}

		ctx.SetUserContext(userCtx)

		span.End()
	}
}

func (t *tracer) Propagator() kit.TracePropagator {
	return t
}

func (t *tracer) Inject(ctx context.Context, carrier kit.TraceCarrier) {
	t.p.Inject(ctx, connCarrier{c: carrier})
}

func (t *tracer) Extract(ctx context.Context, carrier kit.TraceCarrier) context.Context {
	return t.p.Extract(ctx, connCarrier{c: carrier})
}

type connCarrier struct {
	c kit.TraceCarrier
}

func (c connCarrier) Get(key string) string {
	return c.c.Get(key)
}

func (c connCarrier) Set(key string, value string) {
	c.c.Set(key, value)
}

func (c connCarrier) Keys() []string {
	return nil
}
