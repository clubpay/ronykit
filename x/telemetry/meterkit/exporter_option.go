package meterkit

import "go.opentelemetry.io/otel/exporters/prometheus"

type ExporterOption func(e *Exporter) error

func WithPrometheus(_ ...prometheus.Option) ExporterOption {
	return func(e *Exporter) error {
		exp, err := prometheus.New()
		if err != nil {
			return err
		}

		e.rd = exp

		return nil
	}
}
