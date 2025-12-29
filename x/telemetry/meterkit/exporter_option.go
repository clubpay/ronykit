package meterkit

import (
	"fmt"
	"net/http"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
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
		exp, err := prometheus.New(opt...)
		if err != nil {
			return err
		}

		e.rd = exp
		e.shutdownFn = exp.Shutdown

		go func() {
			http.Handle(path, promhttp.HandlerFor(prom.DefaultGatherer, promhttp.HandlerOpts{}))
			err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil) // nosemgrep
			if err != nil {
				panic(err)
			}
		}()

		return nil
	}
}
