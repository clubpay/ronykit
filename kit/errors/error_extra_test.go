package errors_test

import (
	stdErrors "errors"
	"testing"

	"github.com/clubpay/ronykit/kit/errors"
)

func TestNewAndNewG(t *testing.T) {
	err := errors.New("hello %d", 1)
	if err.Error() != "hello 1" {
		t.Fatalf("unexpected New error: %v", err)
	}

	newg := errors.NewG("boom %s")
	err = newg("x")
	if err.Error() != "boom x" {
		t.Fatalf("unexpected NewG error: %v", err)
	}
}

func TestWrapAndIs(t *testing.T) {
	top := stdErrors.New("top")
	down := stdErrors.New("down")

	if errors.Wrap(nil, down) != down {
		t.Fatal("expected wrap with nil top to return down")
	}

	wrapped := errors.Wrap(top, down)
	if !errors.Is(wrapped, top) {
		t.Fatal("expected wrapped error to match top")
	}
	if !errors.Is(wrapped, down) {
		t.Fatal("expected wrapped error to match down")
	}

	unwrapped := stdErrors.Unwrap(wrapped)
	if unwrapped != down {
		t.Fatalf("unexpected unwrap: %v", unwrapped)
	}
}
