package logkit

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

type SugaredLogger struct {
	sz     *zap.SugaredLogger
	prefix string
}

func (l SugaredLogger) Debug(template string, args ...any) {
	l.sz.Debugf(addPrefix(l.prefix, template), args...)
}

func (l SugaredLogger) DebugCtx(ctx context.Context, template string, args ...any) {
	addTraceEvent(ctx, fmt.Sprintf(template, args...))
	l.Debug(template, args...)
}

func (l SugaredLogger) Info(template string, args ...any) {
	l.sz.Infof(addPrefix(l.prefix, template), args...)
}

func (l SugaredLogger) InfoCtx(ctx context.Context, template string, args ...any) {
	addTraceEvent(ctx, fmt.Sprintf(template, args...))
	l.Info(template, args...)
}

func (l SugaredLogger) Warn(template string, args ...any) {
	l.sz.Warnf(addPrefix(l.prefix, template), args...)
}

func (l SugaredLogger) WarnCtx(ctx context.Context, template string, args ...any) {
	addTraceEvent(ctx, fmt.Sprintf(template, args...))
	l.Warn(template, args...)
}

func (l SugaredLogger) Error(template string, args ...any) {
	l.sz.Errorf(addPrefix(l.prefix, template), args...)
}

func (l SugaredLogger) ErrorCtx(ctx context.Context, template string, args ...any) {
	addTraceEvent(ctx, fmt.Sprintf(template, args...))
	l.Error(template, args...)
}

func (l SugaredLogger) Fatal(template string, args ...any) {
	l.sz.Fatalf(addPrefix(l.prefix, template), args...)
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
