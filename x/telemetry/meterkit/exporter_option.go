package meterkit

import (
	"fmt"
	"net/http"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type ExporterOption func(e *Exporter) error

func WithName(name string) ExporterOption {
	return func(e *Exporter) error {
		e.name = name

		return nil
	}
}

func WithPrometheus(path string, port int, opt ...prometheus.Option) ExporterOption {
	return func(e *Exporter) error {
		exp, err := NewPrometheusExporter(opt...)
		if err != nil {
			return err
		}

		e.rd = exp
		e.shutdownFn = exp.Shutdown

		go RunPrometheusServer(path, port)

		return nil
	}
}

func WithMetricReader(rd sdkmetric.Reader) ExporterOption {
	return func(e *Exporter) error {
		e.rd = rd

		return nil
	}
}

func NewPrometheusExporter(opt ...prometheus.Option) (*prometheus.Exporter, error) {
	return prometheus.New(opt...)
}

func RunPrometheusServer(path string, port int) {
	mux := http.NewServeMux()
	mux.Handle(path, promhttp.HandlerFor(prom.DefaultGatherer, promhttp.HandlerOpts{}))
	_ = http.ListenAndServe(fmt.Sprintf(":%d", port), mux) // nosemgrep
}
