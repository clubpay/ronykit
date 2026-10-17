package rkit

import (
	"sync"
	"sync/atomic"
	"time"

	_ "unsafe"
)

/*
   Creation Time: 2020 - Apr - 09
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
*/

var (
	timeInSec atomic.Int64
	clockOnce sync.Once
)

// startClock is deferred to the first TimeUnix call so that importing rkit does not
// spawn a background goroutine.
func startClock() {
	timeInSec.Store(time.Now().Unix())

	go func() {
		for {
			time.Sleep(time.Second)
			timeInSec.Store(time.Now().Unix())
		}
	}()
}

// TimeUnix returns the current Unix time in seconds from a clock refreshed once per second.
// It is cheaper than time.Now().Unix() on hot paths that tolerate up to one second of drift.
func TimeUnix() int64 {
	clockOnce.Do(startClock)

	return timeInSec.Load()
}

func TimeUnixAdd(unixTime int64, d time.Duration) int64 {
	return int64(time.Duration(unixTime) + d/time.Second)
}

func TimeUnixSubtract(unixTime int64, d time.Duration) int64 {
	return int64(time.Duration(unixTime) - d/time.Second)
}

// NanoTime returns the current time in nanoseconds from a monotonic clock.
//
//go:linkname NanoTime runtime.nanotime
func NanoTime() int64

// CPUTicks is a faster alternative to NanoTime to measure time duration.
//
//go:linkname CPUTicks runtime.cputicks
func CPUTicks() int64
