package mem

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/bus/bustest"
	"github.com/clubpay/ronykit/ee/store"
	"github.com/clubpay/ronykit/ee/store/storetest"
)

func TestBusOrderingAndGroups(t *testing.T) {
	b := NewBus(Lanes(4))
	defer b.Close()

	var (
		mu  sync.Mutex
		got = map[string][]string{} // group/key -> payloads in delivery order
		wg  sync.WaitGroup
	)

	handler := func(group string) bus.Handler {
		return func(_ context.Context, m bus.Message) error {
			mu.Lock()
			got[group+"/"+m.PartitionKey] = append(got[group+"/"+m.PartitionKey], string(m.Payload))
			mu.Unlock()
			wg.Done()

			return nil
		}
	}

	ctx := t.Context()
	// Two members in group a share the load; group b gets its own copy.
	for _, g := range []string{"a", "a", "b"} {
		if _, err := b.Subscribe(ctx, g, []string{"e"}, handler(g)); err != nil {
			t.Fatal(err)
		}
	}

	const n = 50

	wg.Add(2 * 2 * n)

	for i := range n {
		for _, k := range []string{"k1", "k2"} {
			_ = b.Publish(ctx, bus.Envelope{ID: fmt.Sprint(k, i), Event: "e", PartitionKey: k, Payload: fmt.Append(nil, i)})
		}
	}

	_ = b.Publish(ctx, bus.Envelope{ID: "x", Event: "unsubscribed"})

	wg.Wait()

	for key, payloads := range got {
		want := make([]string, n)
		for i := range want {
			want[i] = fmt.Sprint(i)
		}

		if !slices.Equal(payloads, want) {
			t.Errorf("%s delivered out of order or incomplete: %v", key, payloads)
		}
	}
}

func TestBusRedeliveryAndRequest(t *testing.T) {
	dead := make(chan bus.Envelope, 1)
	b := NewBus(MaxAttempts(3), RetryBackoff(time.Millisecond), OnDeadLetter(func(e bus.Envelope, _ error) { dead <- e }))

	defer b.Close()

	attempts := 0
	_, _ = b.Subscribe(t.Context(), "g", []string{"fail", "ask"}, func(ctx context.Context, m bus.Message) error {
		if m.Event == "ask" {
			return m.Reply(ctx, bus.Envelope{Payload: append([]byte("re:"), m.Payload...)})
		}

		attempts++

		return errors.New("nope")
	})

	_ = b.Publish(t.Context(), bus.Envelope{ID: "1", Event: "fail"})

	if e := <-dead; e.ID != "1" || attempts != 3 {
		t.Errorf("dead letter %+v after %d attempts", e, attempts)
	}

	r, err := b.Request(t.Context(), bus.Envelope{ID: "2", Event: "ask", Payload: []byte("hi")})
	if err != nil || string(r.Payload) != "re:hi" {
		t.Errorf("reply %q, %v", r.Payload, err)
	}

	if _, err := b.Request(t.Context(), bus.Envelope{ID: "3", Event: "nobody"}); err == nil {
		t.Error("request without subscriber must fail")
	}
}

func TestBusConformance(t *testing.T) {
	bustest.Run(t, func(*testing.T) bus.Bus { return NewBus(RetryBackoff(time.Millisecond)) })
}

func TestConformance(t *testing.T) {
	storetest.Run(t, func(*testing.T) store.Store { return NewStore() })
}
