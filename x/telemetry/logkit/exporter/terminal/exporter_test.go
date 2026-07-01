package terminal

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/log/logtest"
)

func TestExporterExport(t *testing.T) {
	var buf bytes.Buffer
	exp := New(WithWriter(&buf), WithColor(false))

	record := logtest.RecordFactory{
		Timestamp:    time.Date(2026, 7, 1, 15, 4, 5, 0, time.UTC),
		SeverityText: "WARN",
		Body:         log.StringValue("slow query"),
		Attributes: []log.KeyValue{
			log.String("table", "users"),
		},
	}.NewRecord()

	err := exp.Export(context.Background(), []sdklog.Record{record})
	require.NoError(t, err)

	assert.Equal(t,
		"26-07-01T15:04:05 - WARN - - - - - -\nslow query\n<table=users>\n",
		buf.String(),
	)
}

func TestExporterShutdown(t *testing.T) {
	var buf bytes.Buffer
	exp := New(WithWriter(&buf), WithColor(false))

	require.NoError(t, exp.Shutdown(context.Background()))

	record := logtest.RecordFactory{
		Body: log.StringValue("ignored"),
	}.NewRecord()

	err := exp.Export(context.Background(), []sdklog.Record{record})
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}
