package utils

import (
	"errors"
	"strings"
	"testing"
)

func TestPanicErrorHelpers(t *testing.T) {
	base := errors.New("boom")
	err := newPanicError(base)
	pe, ok := err.(*panicError)
	if !ok {
		t.Fatalf("expected panicError, got %T", err)
	}
	if !strings.Contains(pe.Error(), "boom") {
		t.Fatalf("unexpected panic error output: %s", pe.Error())
	}
	if errors.Unwrap(pe) != base {
		t.Fatalf("unexpected unwrap: %v", errors.Unwrap(pe))
	}

	err = newPanicError("plain")
	pe, ok = err.(*panicError)
	if !ok {
		t.Fatalf("expected panicError, got %T", err)
	}
	if errors.Unwrap(pe) != nil {
		t.Fatalf("expected nil unwrap for non-error value")
	}
}
