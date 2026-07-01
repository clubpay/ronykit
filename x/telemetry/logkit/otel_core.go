package logkit

import (
	"reflect"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/log/global"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.uber.org/zap/zapcore"
)

// otelErrorCore wraps the otelzap core so zap.Error fields are emitted as
// OpenTelemetry exception attributes instead of SetErr.
//
// otelzap routes key="error" ErrorType fields through SetErr. When stack traces
// are enabled, otelzap also adds exception.stacktrace before the SDK converts
// the record. The SDK then treats exception.stacktrace as an existing exception
// attribute and skips synthesizing exception.message, so terminal exporters
// that hide stacktrace system attrs lose the error text entirely.
type otelErrorCore struct {
	zapcore.Core
}

func newOtelCore() zapcore.Core {
	return &otelErrorCore{
		Core: otelzap.NewCore("otel", otelzap.WithLoggerProvider(global.GetLoggerProvider())),
	}
}

func (c *otelErrorCore) With(fields []Field) zapcore.Core {
	return &otelErrorCore{Core: c.Core.With(normalizeOtelErrorFields(fields))}
}

func (c *otelErrorCore) Check(ent Entry, ce *CheckedEntry) *CheckedEntry {
	if c.Core.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}

	return ce
}

func (c *otelErrorCore) Write(ent Entry, fields []Field) error {
	return c.Core.Write(ent, normalizeOtelErrorFields(fields))
}

func normalizeOtelErrorFields(fields []Field) []Field {
	if len(fields) == 0 {
		return fields
	}

	out := make([]Field, 0, len(fields)+1)
	for _, f := range fields {
		if f.Type == zapcore.ErrorType && f.Key == "error" {
			err := f.Interface.(error)
			out = append(out,
				String(string(semconv.ExceptionMessageKey), err.Error()),
				String(string(semconv.ExceptionTypeKey), reflect.TypeOf(err).String()),
			)

			continue
		}

		out = append(out, f)
	}

	return out
}
