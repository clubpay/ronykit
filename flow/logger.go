package flow

import (
	"fmt"

	"go.temporal.io/sdk/log"
	"go.uber.org/zap"
)

type zapAdapter struct {
	zl *zap.Logger
}

func NewZapAdapter(zapLogger *zap.Logger) log.Logger {
	return &zapAdapter{
		// Skip one call frame to exclude zap_adapter itself.
		// Or it can be configured when logger is created (not always possible).
		zl: zapLogger.WithOptions(zap.AddCallerSkip(1)),
	}
}

func (log *zapAdapter) fields(keyvals []any) []zap.Field {
	if len(keyvals)%2 != 0 {
		return []zap.Field{zap.Error(fmt.Errorf("odd number of keyvals pairs: %v", keyvals))}
	}

	var fields []zap.Field
	for i := 0; i < len(keyvals); i += 2 {
		key, ok := keyvals[i].(string)
		if !ok {
			key = fmt.Sprintf("%v", keyvals[i])
		}
		fields = append(fields, zap.Any(key, keyvals[i+1]))
	}

	return fields
}

func (log *zapAdapter) Debug(msg string, keyvals ...any) {
	log.zl.Debug(msg, log.fields(keyvals)...)
}

func (log *zapAdapter) Info(msg string, keyvals ...any) {
	log.zl.Info(msg, log.fields(keyvals)...)
}

func (log *zapAdapter) Warn(msg string, keyvals ...any) {
	log.zl.Warn(msg, log.fields(keyvals)...)
}

func (log *zapAdapter) Error(msg string, keyvals ...any) {
	log.zl.Error(msg, log.fields(keyvals)...)
}

func (log *zapAdapter) With(keyvals ...any) log.Logger {
	return &zapAdapter{zl: log.zl.With(log.fields(keyvals)...)}
}

func (log *zapAdapter) WithCallerSkip(skip int) log.Logger {
	return &zapAdapter{zl: log.zl.WithOptions(zap.AddCallerSkip(skip))}
}
