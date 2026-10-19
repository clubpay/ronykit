// Package metrics records ee runtime signals and outbox stats as
// OpenTelemetry metrics. Construct the meter with x/telemetry/meterkit and
// pass it to New.
//
//	exp, err := meterkit.NewExporter(meterkit.WithPrometheus("/metrics", 9464))
//	rec, err := metrics.New(exp.Meter("ee"))
//	env, err := ee.New(ee.WithStore(s), ee.WithBus(b), rec.Option())
//
// Call rec.RecordStats on a timer with env.Stats. The observer must not call
// back into the same Env; RecordStats only writes instruments.
package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/clubpay/ronykit/ee"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	// MetricSignals counts runtime observations. The "kind" attribute is one
	// of the ee.Obs* constants.
	MetricSignals = "ee.runtime.signals"
	// MetricOutboxPending is the unpublished outbox backlog.
	MetricOutboxPending = "ee.outbox.pending"
	// MetricOutboxAge is how long the oldest unpublished entry has waited, in
	// seconds.
	MetricOutboxAge = "ee.outbox.oldest_age_seconds"
)

// Recorder is the ee observer and the outbox gauge writer.
type Recorder struct {
	signals metric.Int64Counter
	pending metric.Int64Gauge
	age     metric.Float64Gauge
}

// New registers the ee instruments on m.
func New(m metric.Meter) (*Recorder, error) {
	signals, err := m.Int64Counter(MetricSignals, metric.WithDescription("ee runtime signals"))
	if err != nil {
		return nil, fmt.Errorf("metrics: signals: %w", err)
	}

	pending, err := m.Int64Gauge(
		MetricOutboxPending,
		metric.WithDescription("unpublished ee outbox entries, including leased ones"),
	)
	if err != nil {
		return nil, fmt.Errorf("metrics: pending: %w", err)
	}

	age, err := m.Float64Gauge(
		MetricOutboxAge,
		metric.WithDescription("age in seconds of the oldest unpublished outbox entry"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("metrics: age: %w", err)
	}

	return &Recorder{signals: signals, pending: pending, age: age}, nil
}

// Option is the ee.WithObserver value for this recorder.
func (r *Recorder) Option() ee.Option {
	return ee.WithObserver(r.Observe)
}

// Observe records one runtime signal. Published and failed outbox batches add
// their count; every other kind adds one.
func (r *Recorder) Observe(ctx context.Context, o ee.Observation) {
	n := int64(1)

	switch o.Kind {
	case ee.ObsOutboxPublished:
		if o.Count == 0 {
			return
		}

		n = int64(o.Count)
	case ee.ObsOutboxError:
		if o.Count > 0 {
			n = int64(o.Count)
		}
	}

	r.signals.Add(ctx, n, metric.WithAttributes(attribute.String("kind", o.Kind)))
}

// RecordStats writes the outbox gauges. A store that cannot report the
// backlog (OutboxPending < 0) leaves the pending gauge unchanged. A zero
// oldest time records age 0 when the backlog is empty, and leaves the age
// unchanged when the backlog is unknown.
func (r *Recorder) RecordStats(ctx context.Context, st ee.Stats) {
	if st.OutboxPending >= 0 {
		r.pending.Record(ctx, int64(st.OutboxPending))
	}

	switch {
	case !st.OutboxOldest.IsZero():
		r.age.Record(ctx, time.Since(st.OutboxOldest).Seconds())
	case st.OutboxPending == 0:
		r.age.Record(ctx, 0)
	}
}
