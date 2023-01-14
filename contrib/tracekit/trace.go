package tracekit

import (
	"context"
	"strings"

	"github.com/clubpay/ronykit/kit"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.11.0"
	"go.opentelemetry.io/otel/trace"
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

var _ kit.Tracer = (*tracer)(nil)

func (t *tracer) Handler() kit.HandlerFunc {
	return func(ctx *kit.Context) {
		userCtx, span := otel.Tracer(t.name).
			Start(
				t.p.Extract(ctx.Context(), envelopeCarrier{e: ctx.In()}),
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

		ctx.AddModifier(
			func(e *kit.Envelope) {
				t.p.Inject(userCtx, envelopeCarrier{e: e})
			},
		)

		ctx.Next()

		span.End()
	}
}

func (t *tracer) Inject(ctx context.Context, carrier kit.TraceCarrier) {
	t.p.Inject(ctx, carrierAdapter{c: carrier})
}

func (t *tracer) Extract(ctx context.Context, carrier kit.TraceCarrier) context.Context {
	return t.p.Extract(ctx, carrierAdapter{c: carrier})
}

type carrierAdapter struct {
	c kit.TraceCarrier
}

func (c carrierAdapter) Get(key string) string {
	return c.c.Get(key)
}

func (c carrierAdapter) Set(key string, value string) {
	c.c.Set(key, value)
}

func (c carrierAdapter) Keys() []string {
	return nil
}

type envelopeCarrier struct {
	e *kit.Envelope
}

func (e envelopeCarrier) Get(key string) string {
	var val string
	e.e.WalkHdr(
		func(k string, v string) bool {
			if strings.EqualFold(key, k) {
				val = v

				return false
			}

			return true
		},
	)

	return val
}

func (e envelopeCarrier) Set(key string, value string) {
	e.e.SetHdr(key, value)
}

func (e envelopeCarrier) Keys() []string {
	return nil
}
