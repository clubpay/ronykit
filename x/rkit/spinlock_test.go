package rkit

import (
	"sync"
	"testing"
)

func TestSpinLock(t *testing.T) {
	const goroutines = 20
	const iterations = 500

	var lock SpinLock
	var wg sync.WaitGroup
	counter := 0

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				lock.Lock()
				counter++
				lock.Unlock()
			}
		}()
	}
	wg.Wait()

	if counter != goroutines*iterations {
		t.Fatalf("counter = %d, want %d", counter, goroutines*iterations)
	}
}
