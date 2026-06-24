package errs_test

import (
	"errors"
	"testing"

	"github.com/clubpay/ronykit/intent/errs"
)

func TestIsNotFound(t *testing.T) {
	if !errs.IsNotFound(errs.SessionNotFound("abc")) {
		t.Fatal("expected session not found")
	}
	if errs.IsNotFound(errs.ErrEmptyPool) {
		t.Fatal("empty pool is not not-found")
	}
	if errs.IsNotFound(nil) {
		t.Fatal("nil is not not-found")
	}
}

func TestWrapNil(t *testing.T) {
	if errs.Wrap(nil, "msg") != nil {
		t.Fatal("expected nil")
	}
}

func TestSentinelsDistinct(t *testing.T) {
	if errors.Is(errs.ErrSessionNotFound, errs.ErrToolNotFound) {
		t.Fatal("sentinels must differ")
	}
}
