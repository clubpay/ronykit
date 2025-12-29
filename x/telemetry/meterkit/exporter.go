package meterkit

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type Exporter struct {
	name       string
	mp         *sdkmetric.MeterProvider
	rd         sdkmetric.Reader
	shutdownFn func(ctx context.Context) error
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

func (exp *Exporter) MeterProvider() metric.MeterProvider {
	return exp.mp
}

func (exp *Exporter) Shutdown(ctx context.Context) error {
	if exp.shutdownFn != nil {
		return exp.shutdownFn(ctx)
	}

	return exp.mp.Shutdown(ctx)
}

func (exp *Exporter) SetAsGlobal() {
	otel.SetMeterProvider(exp.mp)
}
