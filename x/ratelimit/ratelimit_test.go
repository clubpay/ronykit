package ratelimit

import (
	"testing"
	"time"
)

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

func TestRedisInt(t *testing.T) {
	t.Parallel()

	n, err := redisInt(int64(3))
	if err != nil || n != 3 {
		t.Fatalf("int64: got %d, %v", n, err)
	}
	n, err = redisInt(float64(9.7))
	if err != nil || n != 9 {
		t.Fatalf("float64: got %d, %v", n, err)
	}
	n, err = redisInt("4")
	if err != nil || n != 4 {
		t.Fatalf("string: got %d, %v", n, err)
	}
	if _, err := redisInt(true); err == nil {
		t.Fatal("expected error for bool")
	}
}

func TestParseAllowResult_FloatRemaining(t *testing.T) {
	t.Parallel()

	// Mirrors go-redis decoding of allow_n.lua on a successful allow:
	// cost is int64, remaining is float64 from Lua division.
	res, err := parseAllowResult(PerMinute(10), []any{
		int64(1),
		float64(9),
		"-1",
		"6.0",
	})
	if err != nil {
		t.Fatalf("parseAllowResult: %v", err)
	}
	if res.Allowed != 1 || res.Remaining != 9 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.RetryAfter != dur(-1) {
		t.Fatalf("RetryAfter = %v, want -1", res.RetryAfter)
	}
	if res.ResetAfter != 6*time.Second {
		t.Fatalf("ResetAfter = %v, want 6s", res.ResetAfter)
	}
}
