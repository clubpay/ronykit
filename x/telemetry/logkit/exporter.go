package logkit

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
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
	tp        log.LoggerProvider
	tpWrapper func(log.LoggerProvider) log.LoggerProvider
}

func NewExporter(serviceName string, opts ...ExporterOption) (*Exporter, error) {
	t := Exporter{}

	for _, opt := range opts {
		opt(&t)
	}

	var (
		b   sdklog.Exporter
		err error
	)

	switch t.exp {
	case expOTLP, expOTLPHttp:
		b, err = otlpHTTP00Exporter(t.endpoint)
	case expOTLPGrpc:
		b, err = otlpGrpcExporter(t.endpoint)
	case expSTDPretty:
		b, err = stdoutlog.New(stdoutlog.WithPrettyPrint())
	case expSTD:
		b, err = stdoutlog.New()
	default:
		return &t, nil
	}

	if err != nil {
		return nil, err
	}

	t.tp = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(
			sdklog.NewBatchProcessor(b),
		),
		sdklog.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(serviceName),
			),
		),
	)

	if t.tpWrapper != nil {
		global.SetLoggerProvider(t.tpWrapper(t.tp))
	} else {
		global.SetLoggerProvider(t.tp)
	}

	return &t, nil
}

func otlpHTTP00Exporter(endPoint string) (sdklog.Exporter, error) {
	return otlploghttp.New(
		context.Background(),
		otlploghttp.WithEndpoint(endPoint),
		otlploghttp.WithInsecure(),
		otlploghttp.WithCompression(otlploghttp.GzipCompression),
		otlploghttp.WithRetry(
			otlploghttp.RetryConfig{
				Enabled:         true,
				InitialInterval: time.Second,
				MaxInterval:     10 * time.Second,
				MaxElapsedTime:  time.Minute,
			},
		),
	)
}

func otlpGrpcExporter(endPoint string) (sdklog.Exporter, error) {
	return otlploggrpc.New(
		context.Background(),
		otlploggrpc.WithEndpoint(endPoint),
		otlploggrpc.WithInsecure(),
		otlploggrpc.WithCompressor("gzip"),
		otlploggrpc.WithRetry(
			otlploggrpc.RetryConfig{
				Enabled:         true,
				InitialInterval: time.Second,
				MaxInterval:     10 * time.Second,
				MaxElapsedTime:  time.Minute,
			},
		),
	)
}
