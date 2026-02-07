package p

import (
	"testing"
	"time"
)

func TestTimerPoolLifecycle(t *testing.T) {
	timer := AcquireTimer(10 * time.Millisecond)
	if timer == nil {
		t.Fatalf("expected timer")
	}
	ReleaseTimer(timer)

	timer2 := AcquireTimer(5 * time.Millisecond)
	ResetTimer(timer2, 2*time.Millisecond)
	ReleaseTimer(timer2)
}
