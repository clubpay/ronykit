package tracekit

import (
	"strings"

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

func B3(name string, opts ...Option) kit.HandlerFunc {
	cfg := &config{
		tracerName: name,
		propagator: b3Propagator,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return withTracer(cfg)
}

func W3C(name string, opts ...Option) kit.HandlerFunc {
	cfg := &config{
		tracerName: name,
		propagator: w3cPropagator,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return withTracer(cfg)
}

func withTracer(cfg *config) kit.HandlerFunc {
	var (
		traceCtx     propagation.TextMapPropagator
		traceCarrier func(ctx *kit.Context) propagation.TextMapCarrier
	)

	switch cfg.propagator {
	case b3Propagator:
		traceCtx = b3.New(b3.WithInjectEncoding(b3.B3SingleHeader))
		traceCarrier = newB3Carrier
	default:
		traceCtx = propagation.TraceContext{}
		traceCarrier = newW3CCarrier
	}

	var (
		spanOpts []trace.SpanStartOption
		kvs      []attribute.KeyValue
	)

	for k, v := range cfg.tags {
		kvs = append(kvs, attribute.String(k, v))
	}

	if cfg.serviceName != "" {
		kvs = append(kvs, semconv.ServiceNameKey.String(cfg.serviceName))
	}
	if cfg.env != "" {
		kvs = append(kvs, semconv.DeploymentEnvironmentKey.String(cfg.env))
	}

	if len(kvs) > 0 {
		spanOpts = append(spanOpts, trace.WithAttributes(kvs...))
	}

	return func(ctx *kit.Context) {
		userCtx, span := otel.Tracer(cfg.tracerName).
			Start(
				traceCtx.Extract(ctx.Context(), traceCarrier(ctx)),
				ctx.Route(),
				spanOpts...,
			)

		if cfg.dynTags != nil {
			dynTags := cfg.dynTags(ctx.Limited())
			kvs := make([]attribute.KeyValue, 0, len(dynTags))
			for k, v := range cfg.dynTags(ctx.Limited()) {
				kvs = append(kvs, attribute.String(k, v))
			}
			span.SetAttributes(kvs...)
		}

		ctx.SetUserContext(userCtx)
		ctx.Next()

		_, ok := ctx.Conn().(kit.RESTConn)
		if ok {
			span.SetAttributes(semconv.HTTPStatusCodeKey.Int(ctx.GetStatusCode()))
		}

		span.End()
	}
}

type w3cCarrier struct {
	traceParent string
	traceState  string
	ctx         *kit.Context
}

func newW3CCarrier(ctx *kit.Context) propagation.TextMapCarrier {
	c := w3cCarrier{ctx: ctx}
	c.ctx.Conn().Walk(
		func(key string, v string) bool {
			if strings.EqualFold(w3cTraceParent, key) {
				c.traceParent = v
			} else if strings.EqualFold(w3cState, key) {
				c.traceState = v
			}

			return true
		},
	)

	return c
}

func (c w3cCarrier) Get(key string) string {
	switch key {
	case w3cTraceParent:
		return c.traceParent
	case w3cState:
		return c.traceState
	}

	v, ok := c.ctx.Get(key).(string)
	if !ok {
		return ""
	}

	return v
}

func (c w3cCarrier) Set(key string, value string) {
	c.ctx.Set(key, value)
}

func (c w3cCarrier) Keys() []string {
	var keys []string
	c.ctx.Conn().Walk(
		func(key string, _ string) bool {
			keys = append(keys, key)

			return true
		},
	)

	return keys
}

type b3Carrier struct {
	b3           string
	traceID      string
	parentSpanID string
	spanID       string
	sampled      string
	flags        string
	ctx          *kit.Context
}

func newB3Carrier(ctx *kit.Context) propagation.TextMapCarrier {
	c := b3Carrier{ctx: ctx}
	c.ctx.Conn().Walk(
		func(key string, v string) bool {
			switch {
			case strings.EqualFold(b3Single, key):
				c.b3 = v
			case strings.EqualFold(b3TraceID, key):
				c.traceID = v
			case strings.EqualFold(b3SpanID, key):
				c.spanID = v
			case strings.EqualFold(b3ParentSpanID, key):
				c.parentSpanID = v
			case strings.EqualFold(b3Sampled, key):
				c.sampled = v
			case strings.EqualFold(b3Flags, key):
				c.flags = v
			}

			return true
		},
	)

	return c
}

func (c b3Carrier) Get(key string) string {
	switch key {
	case b3Single:
		return c.b3
	case b3TraceID:
		return c.traceID
	case b3SpanID:
		return c.spanID
	case b3ParentSpanID:
		return c.parentSpanID
	case b3Sampled:
		return c.sampled
	case b3Flags:
		return c.flags
	}

	v, ok := c.ctx.Get(key).(string)
	if !ok {
		return ""
	}

	return v
}

func (c b3Carrier) Set(key string, value string) {
	c.ctx.Set(key, value)
}

func (c b3Carrier) Keys() []string {
	var keys []string
	c.ctx.Conn().Walk(
		func(key string, _ string) bool {
			keys = append(keys, key)

			return true
		},
	)

	return keys
}
