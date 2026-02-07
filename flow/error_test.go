package flow

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestWrapError(t *testing.T) {
	if WrapError(context.Background(), nil) != nil {
		t.Fatalf("expected nil for nil error")
	}

	err := WrapError(context.Background(), errors.New("boom"))
	if err == nil {
		t.Fatalf("expected wrapped error")
	}
	if !strings.Contains(err.Error(), "traceID[") {
		t.Fatalf("expected traceID in error: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected original error in message: %s", err.Error())
	}
}
