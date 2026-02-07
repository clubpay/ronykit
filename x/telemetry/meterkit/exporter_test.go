package meterkit

import (
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestExporterOptions(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	exp, err := NewExporter(
		WithName("custom"),
		WithMetricReader(reader),
	)
	if err != nil {
		t.Fatalf("NewExporter error: %v", err)
	}
	if exp.name != "custom" {
		t.Fatalf("name = %q, want %q", exp.name, "custom")
	}
	if exp.rd == nil {
		t.Fatalf("expected reader to be set")
	}
}
