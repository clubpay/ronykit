package log

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx/fxevent"
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
	case "sensitive":
		cores = append(cores,
			zapcore.NewCore(encodeBuilder.SensitiveEncoder(), zapcore.Lock(os.Stdout), l.lvl),
		)
	case "json":
		cores = append(cores,
			zapcore.NewCore(encodeBuilder.JsonEncoder(), zapcore.Lock(os.Stdout), l.lvl),
		)
	case "console":
		cores = append(cores,
			zapcore.NewCore(encodeBuilder.ConsoleEncoder(), zapcore.Lock(os.Stdout), l.lvl),
		)
	}

	core := zapcore.NewTee(append(cores, cfg.cores...)...)
	l.z = zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(FatalLevel),
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

func (l *Logger) Sugared() *SugaredLogger {
	return &SugaredLogger{
		sz:     l.z.Sugar(),
		prefix: l.prefix,
	}
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
	addTraceEvent(ctx, msg, fields...)
	fields = append(fields, zap.String("traceID", Span(ctx).SpanContext().TraceID().String()))

	l.Debug(msg, fields...)
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
	addTraceEvent(ctx, msg, fields...)
	fields = append(fields, zap.String("traceID", Span(ctx).SpanContext().TraceID().String()))

	l.Info(msg, fields...)
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
	addTraceEvent(ctx, msg, fields...)
	fields = append(fields, zap.String("traceID", Span(ctx).SpanContext().TraceID().String()))

	l.Warn(msg, fields...)
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
	addTraceEvent(ctx, msg, fields...)
	fields = append(fields, zap.String("traceID", Span(ctx).SpanContext().TraceID().String()))

	l.Error(msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...Field) {
	if l == nil {
		return
	}
	l.z.Fatal(addPrefix(l.prefix, msg), fields...)
}

func (l *Logger) FatalCtx(ctx context.Context, msg string, fields ...Field) {
	addTraceEvent(ctx, msg, fields...)
	fields = append(fields, zap.String("traceID", Span(ctx).SpanContext().TraceID().String()))

	l.Fatal(msg, fields...)
}

func (l *Logger) RecoverPanic(funcName string, extraInfo interface{}, compensationFunc func()) {
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

func (l *Logger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		l.logEvent("OnStart hook executing",
			zap.String("callee", e.FunctionName),
			zap.String("caller", e.CallerName),
		)
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			l.logError("OnStart hook failed",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.Error(e.Err),
			)
		} else {
			l.logEvent("OnStart hook executed",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.String("runtime", e.Runtime.String()),
			)
		}
	case *fxevent.OnStopExecuting:
		l.logEvent("OnStop hook executing",
			zap.String("callee", e.FunctionName),
			zap.String("caller", e.CallerName),
		)
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			l.logError("OnStop hook failed",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.Error(e.Err),
			)
		} else {
			l.logEvent("OnStop hook executed",
				zap.String("callee", e.FunctionName),
				zap.String("caller", e.CallerName),
				zap.String("runtime", e.Runtime.String()),
			)
		}
	case *fxevent.Supplied:
		if e.Err != nil {
			l.logError("error encountered while applying options",
				zap.String("type", e.TypeName),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.Error(e.Err))
		} else {
			l.logEvent("supplied",
				zap.String("type", e.TypeName),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
			)
		}
	case *fxevent.Provided:
		for _, rtype := range e.OutputTypeNames {
			l.logEvent("provided",
				zap.String("constructor", e.ConstructorName),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.String("type", rtype),
				maybeBool("private", e.Private),
			)
		}
		if e.Err != nil {
			l.logError("error encountered while applying options",
				moduleField(e.ModuleName),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				zap.Error(e.Err))
		}
	case *fxevent.Replaced:
		for _, rtype := range e.OutputTypeNames {
			l.logEvent("replaced",
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.String("type", rtype),
			)
		}
		if e.Err != nil {
			l.logError("error encountered while replacing",
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.Error(e.Err))
		}
	case *fxevent.Decorated:
		for _, rtype := range e.OutputTypeNames {
			l.logEvent("decorated",
				zap.String("decorator", e.DecoratorName),
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.String("type", rtype),
			)
		}
		if e.Err != nil {
			l.logError("error encountered while applying options",
				zap.Strings("stacktrace", e.StackTrace),
				zap.Strings("moduletrace", e.ModuleTrace),
				moduleField(e.ModuleName),
				zap.Error(e.Err))
		}
	case *fxevent.Run:
		if e.Err != nil {
			l.logError("error returned",
				zap.String("name", e.Name),
				zap.String("kind", e.Kind),
				moduleField(e.ModuleName),
				zap.Error(e.Err),
			)
		} else {
			l.logEvent("run",
				zap.String("name", e.Name),
				zap.String("kind", e.Kind),
				zap.String("runtime", e.Runtime.String()),
				moduleField(e.ModuleName),
			)
		}
	case *fxevent.Invoking:
		// Do not log stack as it will make logs hard to read.
		l.logEvent("invoking",
			zap.String("function", e.FunctionName),
			moduleField(e.ModuleName),
		)
	case *fxevent.Invoked:
		if e.Err != nil {
			l.logError("invoke failed",
				zap.Error(e.Err),
				zap.String("stack", e.Trace),
				zap.String("function", e.FunctionName),
				moduleField(e.ModuleName),
			)
		}
	case *fxevent.Stopping:
		l.logEvent("received signal",
			zap.String("signal", strings.ToUpper(e.Signal.String())))
	case *fxevent.Stopped:
		if e.Err != nil {
			l.logError("stop failed", zap.Error(e.Err))
		}
	case *fxevent.RollingBack:
		l.logError("start failed, rolling back", zap.Error(e.StartErr))
	case *fxevent.RolledBack:
		if e.Err != nil {
			l.logError("rollback failed", zap.Error(e.Err))
		}
	case *fxevent.Started:
		if e.Err != nil {
			l.logError("start failed", zap.Error(e.Err))
		} else {
			l.logEvent("started")
		}
	case *fxevent.LoggerInitialized:
		if e.Err != nil {
			l.logError("custom logger initialization failed", zap.Error(e.Err))
		} else {
			l.logEvent("initialized custom fxevent.Logger", zap.String("function", e.ConstructorName))
		}
	}
}

func (l *Logger) logEvent(msg string, fields ...Field) {
	l.z.Log(l.z.Level(), msg, fields...)
}

func (l *Logger) logError(msg string, fields ...Field) {
	lvl := zapcore.WarnLevel
	l.z.Log(lvl, msg, fields...)
}

func Span(ctx context.Context, attrs ...attribute.KeyValue) trace.Span {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)

	return span
}

func moduleField(name string) Field {
	if len(name) == 0 {
		return zap.Skip()
	}
	return zap.String("module", name)
}

func maybeBool(name string, b bool) Field {
	if b {
		return zap.Bool(name, true)
	}
	return zap.Skip()
}
