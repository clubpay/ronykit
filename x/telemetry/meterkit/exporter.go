package meterkit

import (
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Exporter struct {
	name string
	mp   *sdkmetric.MeterProvider
	rd   sdkmetric.Reader
}

func NewExporter(opt ...ExporterOption) (*Exporter, error) {
	exp := &Exporter{
		name: "meterKIT",
	}
	for _, o := range opt {
		err := o(exp)
		if err != nil {
			return nil, err
		}
	}

	exp.mp = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exp.rd),
	)

	return exp, nil
}

func (exp *Exporter) Meter(name string) metric.Meter {
	return exp.mp.Meter(name)
}
