package tracekit

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type ExporterOption func(t *Exporter)

func WithOTLP(endPoint string) ExporterOption {
	return func(t *Exporter) {
		t.exp = expOTLP
		t.endpoint = endPoint
	}
}

func WithHttpOTLP(endPoint string) ExporterOption {
	return func(t *Exporter) {
		t.exp = expOTLPHttp
		t.endpoint = endPoint
	}
}

func WithGrpcOTLP(endPoint string) ExporterOption {
	return func(t *Exporter) {
		t.exp = expOTLPGrpc
		t.endpoint = endPoint
	}
}

func WithTerminal(pretty bool) ExporterOption {
	return func(t *Exporter) {
		if pretty {
			t.exp = expSTDPretty
		} else {
			t.exp = expSTD
		}
	}
}

func WithCustomSampler(s sdktrace.Sampler) ExporterOption {
	return func(t *Exporter) {
		t.sampler = s
	}
}

func WithTracerProviderWrapper(tp func(trace.TracerProvider) trace.TracerProvider) ExporterOption {
	return func(t *Exporter) {
		t.tpWrapper = tp
	}
}
