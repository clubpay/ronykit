package meterkit

import (
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Exporter struct {
	mp *sdkmetric.MeterProvider
	rd sdkmetric.Reader
}

func NewExporter(opt ...ExporterOption) (*Exporter, error) {
	exp := &Exporter{}
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
