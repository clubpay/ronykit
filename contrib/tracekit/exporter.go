package tracekit

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

type exporter string

const (
	expOTLP      exporter = "otlp"
	expOTLPHttp  exporter = "otlp-http"
	expOTLPGrpc  exporter = "otlp-grpc"
	expSTD       exporter = "std"
	expSTDPretty exporter = "std-pretty"
)

type Exporter struct {
	exp       exporter
	endpoint  string
	tp        trace.TracerProvider
	tpWrapper func(trace.TracerProvider) trace.TracerProvider
	sampler   sdktrace.Sampler
}

func NewExporter(serviceName string, opts ...ExporterOption) (*Exporter, error) {
	t := Exporter{
		sampler: sdktrace.ParentBased(defaultSampler),
	}

	for _, opt := range opts {
		opt(&t)
	}

	var (
		b   sdktrace.SpanExporter
		err error
	)

	switch t.exp {
	case expOTLP, expOTLPHttp:
		b, err = otlphttpExporter(t.endpoint)
	case expOTLPGrpc:
		b, err = otlpGrpcExporter(t.endpoint)
	case expSTDPretty:
		b, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	case expSTD:
		b, err = stdouttrace.New()
	default:
		return &t, nil
	}

	if err != nil {
		return nil, err
	}

	t.tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(
			b,
			sdktrace.WithMaxQueueSize(4096),
		),
		sdktrace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(serviceName),
			),
		),
		sdktrace.WithSampler(t.sampler),
	)

	if t.tpWrapper != nil {
		otel.SetTracerProvider(t.tpWrapper(t.tp))
	} else {
		otel.SetTracerProvider(t.tp)
	}

	return &t, nil
}

func otlphttpExporter(endPoint string) (sdktrace.SpanExporter, error) {
	c := otlptracehttp.NewClient(
		otlptracehttp.WithEndpoint(endPoint),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
		otlptracehttp.WithRetry(
			otlptracehttp.RetryConfig{
				Enabled:         true,
				InitialInterval: time.Second,
				MaxInterval:     10 * time.Second,
				MaxElapsedTime:  time.Minute,
			},
		),
	)

	return otlptrace.New(context.Background(), c)
}

func otlpGrpcExporter(endPoint string) (sdktrace.SpanExporter, error) {
	c := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(endPoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithCompressor("gzip"),
		otlptracegrpc.WithRetry(
			otlptracegrpc.RetryConfig{
				Enabled:         true,
				InitialInterval: time.Second,
				MaxInterval:     10 * time.Second,
				MaxElapsedTime:  time.Minute,
			},
		),
	)

	return otlptrace.New(context.Background(), c)
}

type supportShutdown interface {
	Shutdown(ctx context.Context) error
}

func (t Exporter) Shutdown(ctx context.Context) error {
	if t.tp == nil {
		return nil
	}

	if v, ok := t.tp.(supportShutdown); ok {
		return v.Shutdown(ctx)
	}

	return nil
}

var defaultSampler = NewSampler("RonyKit Default Sampler")

type CustomSampler struct {
	desc        string
	dropsByName map[string]struct{}
}

func NewSampler(desc string) *CustomSampler {
	return &CustomSampler{
		desc:        desc,
		dropsByName: make(map[string]struct{}),
	}
}

func (t *CustomSampler) AddDrop(name ...string) *CustomSampler {
	for _, n := range name {
		t.dropsByName[n] = struct{}{}
	}

	return t
}

func (t *CustomSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	_, ok := t.dropsByName[p.Name]
	if ok {
		return sdktrace.SamplingResult{
			Decision:   sdktrace.Drop,
			Tracestate: trace.SpanContextFromContext(p.ParentContext).TraceState(),
		}
	}

	return sdktrace.SamplingResult{
		Decision:   sdktrace.RecordAndSample,
		Tracestate: trace.SpanContextFromContext(p.ParentContext).TraceState(),
	}
}

func (t *CustomSampler) Description() string {
	return t.desc
}
