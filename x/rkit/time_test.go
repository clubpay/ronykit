package rkit

import (
	"testing"
	"time"
)

func TestTimeUnix(t *testing.T) {
	now := time.Now().Unix()
	got := TimeUnix()
	diff := got - now
	if diff < -2 || diff > 2 {
		t.Fatalf("TimeUnix = %d, want within 2s of %d", got, now)
	}
}

func TestTimeUnixMath(t *testing.T) {
	if got := TimeUnixAdd(1000, 2*time.Second); got != 1002 {
		t.Fatalf("TimeUnixAdd = %d, want 1002", got)
	}
	if got := TimeUnixSubtract(1000, 2*time.Second); got != 998 {
		t.Fatalf("TimeUnixSubtract = %d, want 998", got)
	}
}
