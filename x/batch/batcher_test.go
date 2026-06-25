package batch_test

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clubpay/ronykit/x/batch"
	"github.com/stretchr/testify/assert"
)

func TestFlusherWithoutWaitTime(t *testing.T) {
	var out, in int64
	f := batch.NewMulti[int, batch.NA](
		func(targetID string, entries []batch.Entry[int, batch.NA]) {
			time.Sleep(time.Millisecond * 100)
			atomic.AddInt64(&out, int64(len(entries)))
		},
		batch.WithBatchSize(20),
		batch.WithMaxWorkers(10),
	)

	wg := sync.WaitGroup{}
	total := int64(10000)
	for i := 0; i < int(total); i++ {
		wg.Add(1)
		go func() {
			f.EnterAndWait(
				fmt.Sprintf("T%d", rand.Intn(3)),
				batch.NewEntry[int, batch.NA](rand.Intn(10), nil),
			)
			atomic.AddInt64(&in, 1)
			wg.Done()
		}()
	}
	wg.Wait()

	for _, q := range f.Pool() {
		assert.Empty(t, q.EntryChan())
	}
	assert.Equal(t, total, in)
	assert.Equal(t, total, out)
}

func TestFlusherWithWaitTime(t *testing.T) {
	var out, in int64
	f := batch.NewMulti[int, batch.NA](
		func(targetID string, entries []batch.Entry[int, batch.NA]) {
			time.Sleep(time.Millisecond * 100)
			atomic.AddInt64(&out, int64(len(entries)))
			for _, e := range entries {
				e.Value()
			}
		},
		batch.WithBatchSize(20),
		batch.WithMaxWorkers(10),
		batch.WithMinWaitTime(250*time.Millisecond),
	)

	wg := sync.WaitGroup{}
	total := int64(10000)
	for i := 0; i < int(total); i++ {
		wg.Add(1)
		go func() {
			f.EnterAndWait(
				fmt.Sprintf("T%d", rand.Intn(3)),
				batch.NewEntry[int, batch.NA](rand.Intn(10), nil),
			)
			atomic.AddInt64(&in, 1)
			wg.Done()
		}()
	}
	wg.Wait()

	for _, q := range f.Pool() {
		assert.Empty(t, q.EntryChan())
	}
	assert.Equal(t, total, in)
	assert.Equal(t, total, out)
}

func TestFlusherWithCallback(t *testing.T) {
	var out, in int64
	f := batch.NewMulti[int, int](
		func(targetID string, entries []batch.Entry[int, int]) {
			time.Sleep(time.Millisecond * 100)
			atomic.AddInt64(&out, int64(len(entries)))
			for _, e := range entries {
				e.Callback(e.Value())
			}
		},
		batch.WithBatchSize(20),
		batch.WithMaxWorkers(10),
		batch.WithMinWaitTime(250*time.Millisecond),
	)

	wg := sync.WaitGroup{}
	total := int64(10000)
	var sum int64
	for i := 0; i < int(total); i++ {
		wg.Add(1)
		go func(x int) {
			f.EnterAndWait(
				"sameID",
				batch.NewEntry(
					x,
					func(out int) { atomic.AddInt64(&sum, int64(out)) },
				),
			)
			atomic.AddInt64(&in, 1)
			wg.Done()
		}(i)
	}
	wg.Wait()

	for _, q := range f.Pool() {
		assert.Empty(t, q.EntryChan())
	}
	assert.Equal(t, total, in)
	assert.Equal(t, total, out)
	assert.Equal(t, total*(total-1)/2, sum)
}

func ExampleBatcher() {
	averageAll := func(targetID string, entries []batch.Entry[float64, float64]) {
		var (
			sum float64
			n   int
		)
		for _, entry := range entries {
			sum += entry.Value()
			n++
		}
		avg := sum / float64(n)

		for _, e := range entries {
			e.Callback(avg)
		}
	}
	b := batch.NewBatcher(
		averageAll, "tag1",
		batch.WithBatchSize(10),
		batch.WithMinWaitTime(time.Second),
	)
	wg := sync.WaitGroup{}
	for i := 0.0; i < 10.0; i++ {
		wg.Add(1)
		go func(i float64) {
			t := time.Now()
			b.EnterAndWait(
				batch.NewEntry(i, func(out float64) { fmt.Println("duration:", time.Now().Sub(t), "avg:", out) }),
			)
			wg.Done()
		}(i)
	}
	wg.Wait()
}
