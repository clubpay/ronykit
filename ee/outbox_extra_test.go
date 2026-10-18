package ee_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/mem"
	"github.com/clubpay/ronykit/ee/store"
)

// recorder collects envelope IDs delivered by the bus.
type recorder struct {
	mu  sync.Mutex
	ids []string
}

func (r *recorder) handle(_ context.Context, m bus.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ids = append(r.ids, m.ID)

	return nil
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.ids)
}

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return slices.Clone(r.ids)
}

func envJSON(t *testing.T, id, event string) []byte {
	t.Helper()

	data, err := json.Marshal(bus.Envelope{ID: id, Event: event, Payload: []byte("x"), CreatedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func appendOutboxEntries(t *testing.T, st *mem.Store, ctx context.Context, entries ...store.OutboxEntry) {
	t.Helper()

	_ = st.Tx(ctx, func(ctx context.Context, tx store.Tx) error { return tx.AppendOutbox(ctx, entries...) })
}

func leftInStore(st *mem.Store, ctx context.Context) int {
	got, _ := st.Outbox().Claim(ctx, 10, time.Minute)

	_ = st.Outbox().Ack(ctx, idsOf(got)...)

	return len(got)
}

func idsOf(es []store.OutboxEntry) []string {
	out := make([]string, 0, len(es))
	for _, e := range es {
		out = append(out, e.ID)
	}

	return out
}

func TestOutboxDrainsInBatches(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	const total = 120

	events := make([]string, total)
	entries := make([]store.OutboxEntry, total)
	for i := range total {
		events[i] = fmt.Sprintf("batch.%d", i)
		entries[i] = store.OutboxEntry{ID: fmt.Sprintf("b-%d", i), Data: envJSON(t, fmt.Sprintf("b-%d", i), events[i])}
	}

	rec := &recorder{}
	if _, err := f.bus.Subscribe(ctx, "rec", events, rec.handle); err != nil {
		t.Fatal(err)
	}

	appendOutboxEntries(t, f.store, ctx, entries...)

	f.start(t)

	eventually(t, fmt.Sprintf("%d drained envelopes", total), func() bool {
		return rec.count() >= total
	})

	if got := slices.Compact(slices.Sorted(slices.Values(rec.snapshot()))); len(got) != total {
		t.Errorf("delivered %d distinct envelopes, want %d", len(got), total)
	}

	if left := leftInStore(f.store, ctx); left != 0 {
		t.Errorf("%d entries left in the outbox", left)
	}
}

func TestOutboxCorruptEntrySkipped(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	rec := &recorder{}
	if _, err := f.bus.Subscribe(ctx, "rec", []string{"corrupt.good"}, rec.handle); err != nil {
		t.Fatal(err)
	}

	appendOutboxEntries(t, f.store, ctx,
		store.OutboxEntry{ID: "bad", Data: []byte("]{nope")},
		store.OutboxEntry{ID: "good", Data: envJSON(t, "good", "corrupt.good")},
	)

	f.start(t)

	eventually(t, "the good entry", func() bool { return rec.count() >= 1 })

	if got := rec.snapshot(); !slices.Equal(got, []string{"good"}) {
		t.Errorf("delivered %v, want only the good entry", got)
	}

	if left := leftInStore(f.store, ctx); left != 0 {
		t.Errorf("%d entries left: the corrupt entry must be dropped and acked", left)
	}
}

// fakeLeaseStore pairs a fault store with a manual outbox clock, so lease
// expiry can be simulated without waiting for real time.
func fakeLeaseStore() (*faultStore, func(time.Duration)) {
	var cur int64

	atomic.StoreInt64(&cur, time.Unix(0, 0).UnixNano())

	clock := func() time.Time { return time.Unix(0, atomic.LoadInt64(&cur)) }

	st := &faultStore{Store: mem.NewStore(mem.WithStoreClock(clock))}

	return st, func(d time.Duration) { atomic.AddInt64(&cur, int64(d)) }
}

// relayFixture wires an env over a given store and bus with a fast relay tick,
// all its controllers, and test cleanup.
func relayFixture(t *testing.T, st store.Store, bs bus.Bus) *fixture {
	t.Helper()

	return newFaultFixture(t, st, bs, ee.WithOutboxPoll(5*time.Millisecond))
}

func TestOutboxPublishFaultRecovers(t *testing.T) {
	st, advance := fakeLeaseStore()
	bs := &faultBus{Bus: mem.NewBus(mem.RetryBackoff(time.Millisecond))}
	f := relayFixture(t, st, bs)
	ctx := t.Context()

	rec := &recorder{}
	if _, err := bs.Subscribe(ctx, "rec", []string{"pub.fault"}, rec.handle); err != nil {
		t.Fatal(err)
	}

	appendOutboxEntries(t, st.Store, ctx, store.OutboxEntry{ID: "e1", Data: envJSON(t, "e1", "pub.fault")})

	// The first drain fails to publish; the entry stays leased.
	bs.setPubErr(func(int) error { return errStop })

	f.start(t)
	time.Sleep(30 * time.Millisecond)

	// After the (fake) lease expires the entry is claimed and published once.
	advance(31 * time.Second)
	bs.setPubErr(nil)

	eventually(t, "the republished entry", func() bool { return rec.count() >= 1 })

	if got := rec.snapshot(); !slices.Equal(got, []string{"e1"}) {
		t.Errorf("delivered %v, want exactly one e1", got)
	}
	if left := leftInStore(st.Store, ctx); left != 0 {
		t.Errorf("%d entries left", left)
	}
}

func TestOutboxAckFaultDeliversTwice(t *testing.T) {
	st, advance := fakeLeaseStore()
	bs := &faultBus{Bus: mem.NewBus(mem.RetryBackoff(time.Millisecond))}
	f := relayFixture(t, st, bs)
	ctx := t.Context()

	rec := &recorder{}
	if _, err := bs.Subscribe(ctx, "rec", []string{"ack.fault"}, rec.handle); err != nil {
		t.Fatal(err)
	}

	appendOutboxEntries(t, st.Store, ctx, store.OutboxEntry{ID: "e1", Data: envJSON(t, "e1", "ack.fault")})

	// Publish succeeds, but the ack fails: the entry stays leased.
	acked := false
	st.ack = func() error {
		if !acked {
			acked = true

			return errStop
		}

		return nil
	}

	f.start(t)

	eventually(t, "the first delivery", func() bool { return rec.count() >= 1 })

	time.Sleep(20 * time.Millisecond)
	advance(31 * time.Second)

	// The lost ack means the envelope is (correctly) delivered again.
	eventually(t, "the second delivery", func() bool { return rec.count() >= 2 })

	time.Sleep(20 * time.Millisecond)

	if got := rec.snapshot(); !slices.Equal(got, []string{"e1", "e1"}) {
		t.Errorf("delivered %v, want e1 exactly twice (at-least-once)", got)
	}
	if left := leftInStore(st.Store, ctx); left != 0 {
		t.Errorf("%d entries left", left)
	}
}

func TestOutboxClaimFaultRecovers(t *testing.T) {
	st, _ := fakeLeaseStore()
	bs := &faultBus{Bus: mem.NewBus(mem.RetryBackoff(time.Millisecond))}
	f := relayFixture(t, st, bs)
	ctx := t.Context()

	rec := &recorder{}
	if _, err := bs.Subscribe(ctx, "rec", []string{"claim.fault"}, rec.handle); err != nil {
		t.Fatal(err)
	}

	appendOutboxEntries(t, st.Store, ctx, store.OutboxEntry{ID: "e1", Data: envJSON(t, "e1", "claim.fault")})

	claimed := false
	st.claim = func() error {
		if !claimed {
			claimed = true

			return errStop
		}

		return nil
	}

	f.start(t)

	eventually(t, "the recovered claim", func() bool { return rec.count() >= 1 })

	if left := leftInStore(st.Store, ctx); left != 0 {
		t.Errorf("%d entries left", left)
	}
}

func TestOutboxYearClockOverflows(t *testing.T) {
	future := time.Date(20001, 1, 1, 0, 0, 0, 0, time.UTC)

	var (
		mu   sync.Mutex
		dead []string
	)

	bs := mem.NewBus(mem.RetryBackoff(time.Millisecond), mem.MaxAttempts(3),
		mem.OnDeadLetter(func(e bus.Envelope, _ error) {
			mu.Lock()
			dead = append(dead, e.Event)
			mu.Unlock()
		}))
	env := ee.MustNew(ee.WithStore(mem.NewStore()), ee.WithBus(bs),
		ee.WithClock(func() time.Time { return future }),
		ee.WithOutboxPoll(5*time.Millisecond),
		ee.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	orders := must(ee.NewEntityController[Order](env))
	receipts := must(ee.NewEntityController[Receipt](env))
	t.Cleanup(func() { _ = env.Close(); _ = bs.Close() })

	ctx := t.Context()

	_ = orders.Create(ctx, Order{ID: "o1"})

	_ = orders.Action(func(c *ee.ActionCtx[Order, OrderPaid]) error {
		o, err := c.Entity()
		if err != nil {
			return err
		}

		o.Status = "paid"
		if err := c.Save(o); err != nil {
			return err
		}

		// The relay encodes the envelope itself with encoding/json: a clock
		// far outside the representable range fails the commit.
		return c.Emit(ReceiptRequested{OrderID: o.ID})
	})
	if err := env.Start(ctx); err != nil {
		t.Fatal(err)
	}

	_ = env.Send(ctx, OrderPaid{OrderID: "o1"})

	eventually(t, "the dead letter for the overflowing event", func() bool {
		mu.Lock()
		defer mu.Unlock()

		return slices.Equal(dead, []string{"order.paid"})
	})

	if o, _ := orders.Get(ctx, "o1"); o.Status != "" {
		t.Errorf("a unit that failed at commit was applied: %+v", o)
	}
	if _, err := receipts.Get(ctx, "r-o1"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("the emitted event leaked out: %v", err)
	}
}

func TestEnvLifecycle(t *testing.T) {
	ctx := t.Context()

	if _, err := ee.New(); err != nil {
		t.Fatal(err)
	}

	// Starting without backends fails with a clear error.
	storeOnly := ee.MustNew(ee.WithStore(mem.NewStore()))
	if err := storeOnly.Start(ctx); err == nil || !strings.Contains(err.Error(), "bus and a store") {
		t.Errorf("store-only env started: %v", err)
	}

	busOnly := ee.MustNew(ee.WithBus(mem.NewBus()))
	if err := busOnly.Start(ctx); err == nil || !strings.Contains(err.Error(), "bus and a store") {
		t.Errorf("bus-only env started: %v", err)
	}
	_ = busOnly.Close()

	// Registration after Start fails, Close is idempotent and Start stays done.
	bs := mem.NewBus(mem.RetryBackoff(time.Millisecond))
	started := ee.MustNew(ee.WithStore(mem.NewStore()), ee.WithBus(bs),
		ee.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	_ = must(ee.NewEntityController[Order](started))
	_ = must(ee.NewEntityController[Item](started))
	_ = must(ee.NewEntityController[Payment](started))
	if err := started.Start(ctx); err != nil {
		t.Fatal(err)
	}

	if _, err := ee.NewEntityController[Receipt](started); !errors.Is(err, ee.ErrStarted) {
		t.Errorf("register controller after start: %v", err)
	}
	if _, err := ee.NewAggregateController[OrderAgg](started); !errors.Is(err, ee.ErrStarted) {
		t.Errorf("register aggregate after start: %v", err)
	}

	if err := started.Close(); err != nil {
		t.Errorf("first close: %v", err)
	}
	if err := started.Close(); err != nil {
		t.Errorf("second close: %v", err)
	}
	if err := started.Start(ctx); !errors.Is(err, ee.ErrStarted) {
		t.Errorf("start after close: %v", err)
	}
	_ = bs.Close()

	// An action cannot be registered after Start either.
	bs2 := mem.NewBus(mem.RetryBackoff(time.Millisecond))
	late := ee.MustNew(ee.WithStore(mem.NewStore()), ee.WithBus(bs2))
	orders := must(ee.NewEntityController[Order](late))
	if err := late.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := orders.Action(func(*ee.ActionCtx[Order, OrderTouched]) error { return nil }); !errors.Is(err, ee.ErrStarted) {
		t.Errorf("register action after start: %v", err)
	}
	_ = late.Close()
	_ = bs2.Close()
}

func TestStartSubscribeFault(t *testing.T) {
	st := mem.NewStore()
	bs := &faultBus{Bus: mem.NewBus(mem.RetryBackoff(time.Millisecond))}
	env := ee.MustNew(ee.WithStore(st), ee.WithBus(bs))
	orders := must(ee.NewEntityController[Order](env))
	_ = must(ee.NewEntityController[Receipt](env))
	_ = orders.Action(func(*ee.ActionCtx[Order, OrderTouched]) error { return nil })
	t.Cleanup(func() { _ = env.Close(); _ = bs.Close() })

	ctx := t.Context()

	subErrs := 0
	bs.setSubErr(func(string) error {
		if subErrs == 0 {
			subErrs++

			return errStop
		}

		return nil
	})

	if err := env.Start(ctx); err == nil || !errors.Is(err, errStop) ||
		!strings.Contains(err.Error(), "subscribe") {
		t.Fatalf("subscribe fault: %v", err)
	}

	// The failed start closed the partial subscriptions; a retry works.
	if err := env.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if err := env.Close(); err != nil {
		t.Fatal(err)
	}
}
