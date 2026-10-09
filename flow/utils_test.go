package flow

import (
	"testing"

	"go.temporal.io/sdk/temporal"
)

func TestIsApplicationError(t *testing.T) {
	appErr := temporal.NewApplicationError("boom", "test")
	ok, got := IsApplicationError(appErr)
	if !ok {
		t.Fatalf("expected application error")
	}
	if got == nil || got.Error() != appErr.Error() {
		t.Fatalf("unexpected application error: %v", got)
	}

	ok, got = IsApplicationError(temporal.NewCanceledError("cancel"))
	if ok || got != nil {
		t.Fatalf("expected non-application error")
	}
}

func TestErrorHelpers(t *testing.T) {
	appErr := NewApplicationError("boom", "test", "detail")
	ok, got := IsApplicationError(appErr)
	if !ok || got == nil || got.Type() != "test" {
		t.Fatalf("expected application error, got %v", got)
	}

	caused := NewApplicationErrorWithCause("boom", "test", appErr)
	ok, got = IsApplicationError(caused)
	if !ok || got == nil || got.Unwrap() == nil {
		t.Fatalf("expected cause on application error: %v", got)
	}

	nonRetry := NewNonRetryableApplicationError("nope", "bad", nil)
	ok, got = IsApplicationError(nonRetry)
	if !ok || got == nil || !got.NonRetryable() {
		t.Fatalf("expected non-retryable application error")
	}

	withOpts := NewApplicationErrorWithOptions("boom", "typed", ApplicationErrorOptions{NonRetryable: true})
	ok, got = IsApplicationError(withOpts)
	if !ok || got == nil || !got.NonRetryable() {
		t.Fatalf("expected options application error")
	}

	if !IsCanceledError(NewCanceledError("stop")) {
		t.Fatal("expected canceled error")
	}
	if IsCanceledError(appErr) || IsTimeoutError(appErr) || IsTerminatedError(appErr) || IsPanicError(appErr) {
		t.Fatal("application error matched an unrelated predicate")
	}
	if IsWorkflowExecutionAlreadyStartedError(appErr) {
		t.Fatal("application error matched already-started")
	}
}
