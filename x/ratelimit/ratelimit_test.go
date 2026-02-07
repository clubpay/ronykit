package ratelimit

import "testing"

func TestLimitHelpers(t *testing.T) {
	limit := PerSecond(5)
	if limit.Rate != 5 || limit.Burst != 5 {
		t.Fatalf("unexpected limit: %+v", limit)
	}
	if limit.String() != "5 req/s (burst 5)" {
		t.Fatalf("unexpected string: %s", limit.String())
	}

	if !(Limit{}).IsZero() {
		t.Fatalf("expected zero limit")
	}
}

func TestDur(t *testing.T) {
	if got := dur(-1); got != -1 {
		t.Fatalf("dur(-1) = %v, want -1", got)
	}
}
