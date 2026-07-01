package terminal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/sdk/log/logtest"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.opentelemetry.io/otel/trace"
)

func TestFormatRecord(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	require.NoError(t, err)

	ts := time.Date(2026, 7, 1, 15, 4, 5, 0, time.UTC)
	record := logtest.RecordFactory{
		Timestamp:    ts,
		SeverityText: "INFO",
		Body:         log.StringValue("request completed"),
		Attributes: []log.KeyValue{
			log.String("user_id", "42"),
			log.Int("count", 3),
			log.String(string(semconv.CodeFilePathKey), "/app/cmd/service/main.go"),
			log.Int(string(semconv.CodeLineNumberKey), 128),
			log.String(string(semconv.ExceptionMessageKey), "connection refused"),
		},
		TraceID: traceID,
	}.NewRecord()

	got := formatRecord(record, palette{enabled: false})
	want := "26-07-01T15:04:05 - INFO - 4bf92f3577b34da6a3ce929d0e0e4736 - main.go - 128\n" +
		"request completed\n" +
		`<user_id=42>	<count=3>	<error="connection refused">`

	assert.Equal(t, want, got)
}

func TestFormatRecordMissingFields(t *testing.T) {
	record := logtest.RecordFactory{
		Body: log.StringValue("hello"),
	}.NewRecord()

	got := formatRecord(record, palette{enabled: false})
	want := "- - - - - - - - -\nhello\n"

	assert.Equal(t, want, got)
}

func TestFormatAttrLinesWraps(t *testing.T) {
	attrs := []string{
		"<a=1>", "<b=2>", "<c=3>", "<d=4>", "<e=5>", "<f=6>",
	}

	got := formatAttrLines(attrs, palette{enabled: false})
	want := "<a=1>\t<b=2>\t<c=3>\t<d=4>\t<e=5>\n<f=6>"

	assert.Equal(t, want, got)
}

func TestFormatHeaderFields(t *testing.T) {
	record := logtest.RecordFactory{
		Timestamp:    time.Date(2026, 7, 1, 15, 4, 5, 0, time.UTC),
		SeverityText: "WARN",
		Body:         log.StringValue("slow query"),
	}.NewRecord()

	got := formatHeader(record, collectMeta(record), palette{enabled: false})
	assert.Equal(t, "26-07-01T15:04:05 - WARN - - - - - -", got)
}

func TestQuoteValue(t *testing.T) {
	assert.Equal(t, `hello`, quoteValue("hello"))
	assert.Equal(t, `"hello world"`, quoteValue("hello world"))
	assert.Equal(t, `""`, quoteValue(""))
}
