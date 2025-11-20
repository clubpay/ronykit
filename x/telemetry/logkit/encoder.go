package logkit

import (
	"time"

	"go.uber.org/zap/zapcore"
)

type EncoderBuilder struct {
	cfg zapcore.EncoderConfig
}

func NewEncoderBuilder() *EncoderBuilder {
	return &EncoderBuilder{
		cfg: zapcore.EncoderConfig{
			TimeKey:        "",
			LevelKey:       "",
			NameKey:        "",
			CallerKey:      "",
			MessageKey:     "",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeTime:     timeEncoder,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}
}

func (eb *EncoderBuilder) WithTimeKey(key string) *EncoderBuilder {
	eb.cfg.TimeKey = key

	return eb
}

func (eb *EncoderBuilder) WithLevelKey(key string) *EncoderBuilder {
	eb.cfg.LevelKey = key

	return eb
}

func (eb *EncoderBuilder) WithNameKey(key string) *EncoderBuilder {
	eb.cfg.NameKey = key

	return eb
}

func (eb *EncoderBuilder) WithMessageKey(key string) *EncoderBuilder {
	eb.cfg.MessageKey = key

	return eb
}

func (eb *EncoderBuilder) WithCallerKey(key string) *EncoderBuilder {
	eb.cfg.CallerKey = key

	return eb
}

func (eb *EncoderBuilder) ConsoleEncoder() Encoder {
	return zapcore.NewConsoleEncoder(eb.cfg)
}

func (eb *EncoderBuilder) SensitiveEncoder() Encoder {
	return newSensitive(sensitiveConfig{
		EncoderConfig: eb.cfg,
	})
}

func (eb *EncoderBuilder) JsonEncoder() Encoder {
	return zapcore.NewJSONEncoder(eb.cfg)
}

func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("06-01-02T15:04:05"))
}
