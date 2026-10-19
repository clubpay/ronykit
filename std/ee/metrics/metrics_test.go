package metrics_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/mem"
	"github.com/clubpay/ronykit/std/ee/metrics"
	"github.com/clubpay/ronykit/x/telemetry/meterkit"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type order struct {
	ID     string `ee:"id"`
	Status string
}

func (order) EntityName() string { return "order" }

func TestRecorder(t *testing.T) {
	reader := metric.NewManualReader()
	exp, err := meterkit.NewExporter(meterkit.WithMetricReader(reader))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = exp.Shutdown(t.Context()) })

	rec, err := metrics.New(exp.Meter("ee"))
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	rec.Observe(ctx, ee.Observation{Kind: ee.ObsPanic})
	rec.Observe(ctx, ee.Observation{Kind: ee.ObsOutboxPublished, Count: 4})
	rec.Observe(ctx, ee.Observation{Kind: ee.ObsOutboxError, Count: 2})
	rec.RecordStats(ctx, ee.Stats{
		OutboxPending: 3,
		OutboxOldest:  time.Now().Add(-5 * time.Second),
	})

	rm := collect(t, reader)
	if got := sum(rm, metrics.MetricSignals, ee.ObsPanic); got != 1 {
		t.Errorf("panics = %d", got)
	}

	if got := sum(rm, metrics.MetricSignals, ee.ObsOutboxPublished); got != 4 {
		t.Errorf("published = %d", got)
	}

	if got := sum(rm, metrics.MetricSignals, ee.ObsOutboxError); got != 2 {
		t.Errorf("outbox errors = %d", got)
	}

	if got := gauge(rm, metrics.MetricOutboxPending); got != 3 {
		t.Errorf("pending = %d", got)
	}

	if got := floatGauge(rm, metrics.MetricOutboxAge); got < 4 || got > 30 {
		t.Errorf("age = %v", got)
	}

	rec.RecordStats(ctx, ee.Stats{OutboxPending: 0})

	rm = collect(t, reader)
	if got := floatGauge(rm, metrics.MetricOutboxAge); got != 0 {
		t.Errorf("empty age = %v", got)
	}
}

func TestObserverSeesConflicts(t *testing.T) {
	reader := metric.NewManualReader()
	exp, err := meterkit.NewExporter(meterkit.WithMetricReader(reader))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = exp.Shutdown(t.Context()) })

	rec, err := metrics.New(exp.Meter("ee"))
	if err != nil {
		t.Fatal(err)
	}

	env := ee.MustNew(
		ee.WithStore(mem.NewStore()),
		ee.WithBus(mem.NewBus()),
		ee.WithMaxRetries(0),
		rec.Option(),
	)
	orders, err := ee.NewEntityController[order](env)
	if err != nil {
		t.Fatal(err)
	}

	ctx := t.Context()
	if err := orders.Create(ctx, order{ID: "o1", Status: "open"}); err != nil {
		t.Fatal(err)
	}

	var (
		wg   sync.WaitGroup
		read sync.WaitGroup
	)

	read.Add(2)

	for range 2 {
		wg.Go(func() {
			_ = orders.Update(ctx, "o1", func(o *order) error {
				read.Done()
				read.Wait()
				o.Status = "paid"

				return nil
			})
		})
	}

	wg.Wait()

	if got := sum(collect(t, reader), metrics.MetricSignals, ee.ObsConflict); got < 1 {
		t.Fatalf("conflicts = %d", got)
	}
}

func collect(t *testing.T, reader *metric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}

	return rm
}

func sum(rm metricdata.ResourceMetrics, name, kind string) int64 {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}

			data, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}

			var n int64

			for _, p := range data.DataPoints {
				if kindOf(p.Attributes) == kind {
					n += p.Value
				}
			}

			return n
		}
	}

	return 0
}

func gauge(rm metricdata.ResourceMetrics, name string) int64 {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}

			data, ok := m.Data.(metricdata.Gauge[int64])
			if !ok || len(data.DataPoints) == 0 {
				return 0
			}

			return data.DataPoints[len(data.DataPoints)-1].Value
		}
	}

	return 0
}

func floatGauge(rm metricdata.ResourceMetrics, name string) float64 {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}

			data, ok := m.Data.(metricdata.Gauge[float64])
			if !ok || len(data.DataPoints) == 0 {
				return 0
			}

			return data.DataPoints[len(data.DataPoints)-1].Value
		}
	}

	return 0
}

func kindOf(set attribute.Set) string {
	v, _ := set.Value(attribute.Key("kind"))

	return v.AsString()
}
