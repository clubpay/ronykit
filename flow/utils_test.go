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
