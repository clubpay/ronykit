package logkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
	"go.uber.org/zap/zapcore"
)

func TestNormalizeOtelErrorFields(t *testing.T) {
	t.Parallel()

	err := assert.AnError
	got := normalizeOtelErrorFields([]Field{
		{Key: "user", Type: zapcore.StringType, String: "42"},
		Error(err),
	})

	require.Len(t, got, 3)
	assert.Equal(t, "user", got[0].Key)
	assert.Equal(t, string(semconv.ExceptionMessageKey), got[1].Key)
	assert.Equal(t, err.Error(), got[1].String)
	assert.Equal(t, string(semconv.ExceptionTypeKey), got[2].Key)
	assert.NotEmpty(t, got[2].String)
}

func TestNormalizeOtelErrorFieldsNamedError(t *testing.T) {
	t.Parallel()

	err := assert.AnError
	got := normalizeOtelErrorFields([]Field{
		NamedError("db", err),
	})

	require.Len(t, got, 1)
	assert.Equal(t, zapcore.ErrorType, got[0].Type)
	assert.Equal(t, "db", got[0].Key)
}
