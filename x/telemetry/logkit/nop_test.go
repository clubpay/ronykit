package logkit

import (
	"context"
	"log/slog"
	"testing"
)

func TestNopLoggerSLogDoesNotPanic(t *testing.T) {
	nop := newNOP()
	l := nop.SLog()

	if l.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatal("expected nop slog handler to disable debug")
	}
	if l.Enabled(context.Background(), slog.LevelError) {
		t.Fatal("expected nop slog handler to disable error")
	}

	l.Debug("debug")
	l.Info("info")
	l.Warn("warn")
	l.Error("error")
}

func TestNopLoggerSetLevelDoesNotPanic(t *testing.T) {
	newNOP().SetLevel(DebugLevel)
}
