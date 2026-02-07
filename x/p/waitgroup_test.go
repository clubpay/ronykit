package p

import "testing"

func TestWaitGroupPool(t *testing.T) {
	wg := AcquireWaitGroup()
	wg.Add(1)
	wg.Done()
	ReleaseWaitGroup(wg)
}
