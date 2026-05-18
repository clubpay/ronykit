package logkit

import (
	"context"
	"log/slog"

	"go.uber.org/zap/zapcore"
)

var _ slog.Handler = (*slogHandler)(nil)

type slogHandler struct {
	l     *Logger
	attrs []slog.Attr
}

func (s slogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return s.l.lvl.Enabled(lvlFromSlog(level))
}

func (s slogHandler) Handle(ctx context.Context, record slog.Record) error {
	ce := s.l.Check(lvlFromSlog(record.Level), record.Message)
	if ce == nil {
		return nil
	}

	fields := make([]Field, 0, record.NumAttrs()+len(s.attrs))
	record.Attrs(func(attr slog.Attr) bool {
		fields = attrToFieldPrefix(attr, "", fields)

		return true
	})

	ce.Write(getFields(ctx, fields...)...)

	return nil
}

func attrToFieldPrefix(attr slog.Attr, prefix string, fields []Field) []Field {
	switch attr.Value.Kind() {
	default:
		attr.Key = prefix + attr.Key
		fields = append(fields, attrToField(attr))
	case slog.KindGroup:
		for _, subAttr := range attr.Value.Group() {
			if prefix == "" {
				fields = attrToFieldPrefix(subAttr, subAttr.Key+".", fields)
			} else {
				fields = attrToFieldPrefix(subAttr, prefix+subAttr.Key, fields)
			}
		}
	case slog.KindLogValuer:
		fields = attrToFieldPrefix(slog.Attr{Key: attr.Key, Value: attr.Value.Resolve()}, prefix, fields)
	}

	return fields
}

func attrToField(attr slog.Attr) Field {
	switch attr.Value.Kind() {
	case slog.KindAny:
		return Any(attr.Key, attr.Value.Any())
	case slog.KindBool:
		return Bool(attr.Key, attr.Value.Bool())
	case slog.KindDuration:
		return Duration(attr.Key, attr.Value.Duration())
	case slog.KindFloat64:
		return Float64(attr.Key, attr.Value.Float64())
	case slog.KindInt64:
		return Int64(attr.Key, attr.Value.Int64())
	case slog.KindString:
		return String(attr.Key, attr.Value.String())
	case slog.KindUint64:
		return Uint64(attr.Key, attr.Value.Uint64())
	case slog.KindTime:
		return Time(attr.Key, attr.Value.Time())
	default:
		panic("unknown slog kind")
	}
}

func (s slogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return slogHandler{
		l:     s.l,
		attrs: attrs,
	}
}

func (s slogHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return s
	}

	return slogHandler{
		l: s.l.With(name),
	}
}

func lvlFromSlog(lvl slog.Level) Level {
	out, ok := sLogLevels[lvl]
	if !ok {
		return InfoLevel
	}

	return out
}

var sLogLevels = map[slog.Level]zapcore.Level{
	slog.LevelDebug: DebugLevel,
	slog.LevelInfo:  InfoLevel,
	slog.LevelWarn:  WarnLevel,
	slog.LevelError: ErrorLevel,
}
