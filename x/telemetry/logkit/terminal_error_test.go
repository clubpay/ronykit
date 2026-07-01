package logkit_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/clubpay/ronykit/x/telemetry/logkit"
	"github.com/clubpay/ronykit/x/telemetry/logkit/exporter/terminal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type bufExporter struct{ buf bytes.Buffer }

func (b *bufExporter) Export(_ context.Context, records []sdklog.Record) error {
	exp := terminal.New(terminal.WithWriter(&b.buf), terminal.WithColor(false))
	return exp.Export(context.Background(), records)
}

func (b *bufExporter) Shutdown(context.Context) error   { return nil }
func (b *bufExporter) ForceFlush(context.Context) error { return nil }

func TestTerminalExporterShowsErrorWithStacktrace(t *testing.T) {
	var buf bufExporter
	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(&buf)),
		sdklog.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String("test"))),
	)
	global.SetLoggerProvider(lp)

	log := logkit.New()
	log.Error("failed", logkit.Error(errors.New("connection refused")))
	require.NoError(t, log.Sync())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, lp.ForceFlush(ctx))

	assert.Contains(t, buf.buf.String(), "connection refused")
}
