package logkit

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/log/global"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a wrapper around zap.Logger and adds a good few features to it.
// It provides layered logs which could be used by separate packages, and could be turned off or on
// separately. Separate layers could also have independent log levels.
// Whenever you change log level it propagates through its children.
type Logger struct {
	prefix     string
	skipCaller int
	z          *zap.Logger
	otel       *otelzap.Core
	lvl        zap.AtomicLevel
}

func New(opts ...Option) *Logger {
	encodeBuilder := NewEncoderBuilder().
		WithTimeKey("ts").
		WithLevelKey("level").
		WithNameKey("name").
		WithCallerKey("caller").
		WithMessageKey("msg")

	cfg := defaultConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	l := &Logger{
		lvl:        zap.NewAtomicLevelAt(cfg.level),
		skipCaller: cfg.skipCaller,
	}

	var cores []Core

	switch cfg.encoder {
	case "json":
		cores = append(cores,
			zapcore.NewCore(encodeBuilder.JsonEncoder(), zapcore.Lock(os.Stdout), l.lvl),
		)
	case "console":
		cores = append(cores,
			zapcore.NewCore(encodeBuilder.ConsoleEncoder(), zapcore.Lock(os.Stdout), l.lvl),
		)
	}

	l.otel = otelzap.NewCore("otel", otelzap.WithLoggerProvider(global.GetLoggerProvider()))
	cores = append(cores, l.otel)
	core := zapcore.NewTee(
		append(cores, cfg.cores...)...,
	)
	l.z = zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(ErrorLevel),
		zap.AddCallerSkip(cfg.skipCaller),
		zap.Hooks(cfg.hooks...),
	)

	return l
}

func newNOP() *Logger {
	l := &Logger{}
	l.z = zap.NewNop()

	return l
}

func (l *Logger) Sync() error {
	return l.z.Sync()
}

func (l *Logger) SetLevel(lvl Level) {
	l.lvl.SetLevel(lvl)
}

func (l *Logger) With(name string) *Logger {
	return l.WithSkip(name, l.skipCaller)
}

func (l *Logger) WithSkip(name string, skipCaller int) *Logger {
	return l.with(l.z.Core(), name, skipCaller)
}

func (l *Logger) WithCore(core Core) *Logger {
	return l.with(
		zapcore.NewTee(
			l.z.Core(), core,
		),
		"",
		l.skipCaller,
	)
}

func (l *Logger) with(core zapcore.Core, name string, skip int) *Logger {
	prefix := l.prefix
	if name != "" {
		prefix = fmt.Sprintf("%s[%s]", l.prefix, name)
	}

	childLogger := &Logger{
		prefix:     prefix,
		skipCaller: l.skipCaller,
		z: zap.New(
			core,
			zap.AddCaller(),
			zap.AddStacktrace(ErrorLevel),
			zap.AddCallerSkip(skip),
		),
		lvl: l.lvl,
	}

	return childLogger
}

func (l *Logger) WarnOnErr(guideTxt string, err error, fields ...Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
		l.Warn(guideTxt, fields...)
	}
}

func (l *Logger) ErrorOnErr(guideTxt string, err error, fields ...Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
		l.Error(guideTxt, fields...)
	}
}

func (l *Logger) checkLevel(lvl Level) bool {
	if l == nil {
		return false
	}

	// Check the level first to reduce the cost of disabled log calls.
	// Since Panic and higher may exit, we skip the optimization for those levels.
	if lvl < zapcore.DPanicLevel && !l.z.Core().Enabled(lvl) {
		return false
	}

	return true
}

func (l *Logger) Check(lvl Level, msg string) *CheckedEntry {
	if !l.checkLevel(lvl) {
		return nil
	}

	return l.z.Check(lvl, addPrefix(l.prefix, msg))
}

func (l *Logger) Debug(msg string, fields ...Field) {
	if l == nil {
		return
	}

	if !l.checkLevel(DebugLevel) {
		return
	}

	if ce := l.z.Check(DebugLevel, addPrefix(l.prefix, msg)); ce != nil {
		ce.Write(fields...)
	}
}

func (l *Logger) DebugCtx(ctx context.Context, msg string, fields ...Field) {
	l.Debug(msg, getFields(ctx, fields...)...)
}

func (l *Logger) Info(msg string, fields ...Field) {
	if l == nil {
		return
	}

	if !l.checkLevel(InfoLevel) {
		return
	}

	if ce := l.z.Check(InfoLevel, addPrefix(l.prefix, msg)); ce != nil {
		ce.Write(fields...)
	}
}

func (l *Logger) InfoCtx(ctx context.Context, msg string, fields ...Field) {
	l.Info(msg, getFields(ctx, fields...)...)
}

func (l *Logger) Warn(msg string, fields ...Field) {
	if l == nil {
		return
	}

	if !l.checkLevel(WarnLevel) {
		return
	}

	if ce := l.z.Check(WarnLevel, addPrefix(l.prefix, msg)); ce != nil {
		ce.Write(fields...)
	}
}

func (l *Logger) WarnCtx(ctx context.Context, msg string, fields ...Field) {
	l.Warn(msg, getFields(ctx, fields...)...)
}

func (l *Logger) Error(msg string, fields ...Field) {
	if l == nil {
		return
	}

	if !l.checkLevel(ErrorLevel) {
		return
	}

	if ce := l.z.Check(ErrorLevel, addPrefix(l.prefix, msg)); ce != nil {
		ce.Write(fields...)
	}
}

func (l *Logger) ErrorCtx(ctx context.Context, msg string, fields ...Field) {
	l.Error(msg, getFields(ctx, fields...)...)
}

func (l *Logger) Fatal(msg string, fields ...Field) {
	if l == nil {
		return
	}

	l.z.Fatal(addPrefix(l.prefix, msg), fields...)
}

func (l *Logger) FatalCtx(ctx context.Context, msg string, fields ...Field) {
	l.Fatal(msg, getFields(ctx, fields...)...)
}

func (l *Logger) RecoverPanic(funcName string, extraInfo any, compensationFunc func()) {
	if r := recover(); r != nil {
		l.Error("Panic Recovered",
			zap.String("Task", funcName),
			zap.Any("Info", extraInfo),
			zap.Any("Recover", r),
			zap.ByteString("StackTrace", debug.Stack()),
		)

		if compensationFunc != nil {
			go compensationFunc()
		}
	}
}

func addPrefix(prefix, in string) (out string) {
	if prefix != "" {
		sb := &strings.Builder{}
		sb.WriteString(prefix)
		sb.WriteRune(' ')
		sb.WriteString(in)
		out = sb.String()

		return out
	}

	return in
}

func getFields(ctx context.Context, fields ...Field) []zap.Field {
	fields = append(fields, Any("context", ctx))

	return fields
}
