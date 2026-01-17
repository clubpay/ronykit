package logkit

import (
	"go.opentelemetry.io/otel/log"
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

func WithLoggerProviderWrapper(tp func(provider log.LoggerProvider) log.LoggerProvider) ExporterOption {
	return func(t *Exporter) {
		t.tpWrapper = tp
	}
}
