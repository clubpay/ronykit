package flow

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestZapAdapterFields(t *testing.T) {
	adapter := &zapAdapter{zl: zap.NewNop()}
	fields := adapter.fields([]any{"k1", "v1", "k2", 2})
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	if fields[0].Key != "k1" || fields[1].Key != "k2" {
		t.Fatalf("unexpected field keys: %v", []string{fields[0].Key, fields[1].Key})
	}

	odd := adapter.fields([]any{"k1"})
	if len(odd) != 1 || odd[0].Type != zapcore.ErrorType {
		t.Fatalf("expected error field for odd keyvals")
	}
}
