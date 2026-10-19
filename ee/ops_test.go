package ee_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/mem"
	"github.com/clubpay/ronykit/ee/store"
)

func TestCheckAndOptions(t *testing.T) {
	env := ee.MustNew()
	if err := env.Check(); err == nil {
		t.Fatal("Check must reject a missing bus and store")
	}

	if _, err := ee.New(ee.WithOutboxBatch(0)); err == nil {
		t.Fatal("batch 0")
	}

	if _, err := ee.New(ee.WithOutboxLease(0)); err == nil {
		t.Fatal("lease 0")
	}

	if _, err := ee.New(ee.WithAutoSave(), ee.WithRequireSave()); err == nil {
		t.Fatal("auto-save and require-save")
	}
}

func TestRequireSave(t *testing.T) {
	dead := make(chan error, 1)
	b := mem.NewBus(mem.MaxAttempts(1), mem.OnDeadLetter(func(_ bus.Envelope, err error) {
		dead <- err
	}))
	st := mem.NewStore()
	env := ee.MustNew(
		ee.WithStore(st),
		ee.WithBus(b),
		ee.WithRequireSave(),
		ee.WithOutboxPoll(time.Hour),
	)
	orders := must(ee.NewEntityController[Order](env))
	t.Cleanup(func() { _ = env.Close(); _ = b.Close() })

	ctx := t.Context()
	if err := orders.Action(func(ctx *ee.ActionCtx[Order, OrderPaid]) error {
		o, err := ctx.Entity()
		if err != nil {
			return err
		}

		o.Status = "paid"

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := orders.Create(ctx, Order{ID: "o1", Status: "open"}); err != nil {
		t.Fatal(err)
	}

	if err := env.Start(ctx); err != nil {
		t.Fatal(err)
	}

	if err := env.Send(ctx, OrderPaid{OrderID: "o1"}); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-dead:
		if !strings.Contains(err.Error(), "without Save") {
			t.Fatalf("dead letter: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for the unsaved-change error")
	}

	o, err := orders.Get(ctx, "o1")
	if err != nil || o.Status != "open" {
		t.Fatalf("status %q, %v", o.Status, err)
	}
}

func TestConflictObserved(t *testing.T) {
	var (
		mu sync.Mutex
		n  int
	)

	st := newFaultStore(func(s *faultStore) {
		s.txOp = func(op, kind, id string) error {
			if op == "update" {
				return store.ErrConflict
			}

			return nil
		}
	})
	env := ee.MustNew(
		ee.WithStore(st),
		ee.WithBus(mem.NewBus()),
		ee.WithMaxRetries(1),
		ee.WithObserver(func(_ context.Context, o ee.Observation) {
			if o.Kind != ee.ObsConflict {
				return
			}

			mu.Lock()
			n++
			mu.Unlock()
		}),
	)
	orders := must(ee.NewEntityController[Order](env))
	ctx := t.Context()

	if err := orders.Create(ctx, Order{ID: "o1"}); err != nil {
		t.Fatal(err)
	}

	err := orders.Update(ctx, "o1", func(o *Order) error {
		o.Status = "paid"

		return nil
	})
	if !errors.Is(err, ee.ErrConflict) || !strings.Contains(err.Error(), "after 2 attempts") {
		t.Fatalf("update: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if n != 2 {
		t.Fatalf("conflict observations = %d", n)
	}
}

func TestPanicIsRedeliveredSignal(t *testing.T) {
	var kinds []string

	env := ee.MustNew(
		ee.WithStore(mem.NewStore()),
		ee.WithBus(mem.NewBus()),
		ee.WithObserver(func(_ context.Context, o ee.Observation) { kinds = append(kinds, o.Kind) }),
	)
	orders := must(ee.NewEntityController[Order](env))
	ctx := t.Context()

	if err := orders.Create(ctx, Order{ID: "o1"}); err != nil {
		t.Fatal(err)
	}

	err := orders.Update(ctx, "o1", func(*Order) error { panic("boom") })
	if err == nil || !strings.Contains(err.Error(), "panic: boom") {
		t.Fatalf("update: %v", err)
	}

	if len(kinds) != 1 || kinds[0] != ee.ObsPanic {
		t.Fatalf("observations %v", kinds)
	}
}

func TestRequestReply(t *testing.T) {
	f := newFixture(t)

	if err := f.orders.Action(func(ctx *ee.ActionCtx[Order, OrderPaid]) error {
		return ctx.Reply(Receipt{ID: "r1", OrderID: ctx.Event().OrderID})
	}); err != nil {
		t.Fatal(err)
	}

	f.start(t)

	got, err := ee.Request[Receipt](t.Context(), f.env, OrderPaid{OrderID: "o1", Merchant: "m"})
	if err != nil {
		t.Fatal(err)
	}

	if got.ID != "r1" || got.OrderID != "o1" {
		t.Fatalf("reply %+v", got)
	}

	env := ee.MustNew(ee.WithStore(mem.NewStore()), ee.WithBus(bareBus{}))
	if _, err := ee.Request[Receipt](t.Context(), env, OrderPaid{OrderID: "o1"}); !errors.Is(err, ee.ErrNotSupported) {
		t.Fatalf("bare bus: %v", err)
	}
}

func TestReplyWithoutRequest(t *testing.T) {
	dead := make(chan error, 1)
	b := mem.NewBus(mem.MaxAttempts(1), mem.RetryBackoff(time.Millisecond), mem.OnDeadLetter(func(_ bus.Envelope, err error) {
		dead <- err
	}))
	env := ee.MustNew(ee.WithStore(mem.NewStore()), ee.WithBus(b), ee.WithOutboxPoll(time.Hour))
	orders := must(ee.NewEntityController[Order](env))
	t.Cleanup(func() { _ = env.Close(); _ = b.Close() })

	ctx := t.Context()
	if err := orders.Action(func(ctx *ee.ActionCtx[Order, OrderPaid]) error {
		return ctx.Reply(Receipt{ID: "r"})
	}); err != nil {
		t.Fatal(err)
	}

	if err := env.Start(ctx); err != nil {
		t.Fatal(err)
	}

	if err := env.Send(ctx, OrderPaid{OrderID: "o1"}); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-dead:
		if !errors.Is(err, ee.ErrNotSupported) {
			t.Fatalf("dead letter: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out")
	}
}

func TestStatsAndExpireDedup(t *testing.T) {
	st := mem.NewStore()
	b := mem.NewBus()
	env := ee.MustNew(ee.WithStore(st), ee.WithBus(b))
	t.Cleanup(func() { _ = b.Close() })

	ctx := t.Context()
	old := time.Now().Add(-2 * time.Hour).UTC()
	fresh := time.Now().UTC()

	oldData, err := json.Marshal(struct {
		At time.Time `json:"at"`
	}{At: old})
	if err != nil {
		t.Fatal(err)
	}

	freshData, err := json.Marshal(struct {
		At time.Time `json:"at"`
	}{At: fresh})
	if err != nil {
		t.Fatal(err)
	}

	err = st.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
		if err := tx.Insert(ctx, store.Record{Kind: ee.DedupKind, ID: "old", Version: 1, Data: oldData}); err != nil {
			return err
		}

		if err := tx.Insert(ctx, store.Record{Kind: ee.DedupKind, ID: "fresh", Version: 1, Data: freshData}); err != nil {
			return err
		}

		if err := tx.Insert(ctx, store.Record{Kind: ee.DedupKind, ID: "bad", Version: 1, Data: []byte(`{}`)}); err != nil {
			return err
		}

		return tx.AppendOutbox(ctx, store.OutboxEntry{ID: "e1", CreatedAt: old, Data: []byte(`{}`)})
	})
	if err != nil {
		t.Fatal(err)
	}

	n, err := env.ExpireDedup(ctx, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	if n != 2 {
		t.Fatalf("expired %d, want old and corrupt", n)
	}

	if st.Len(ee.DedupKind) != 1 {
		t.Fatalf("markers left: %d", st.Len(ee.DedupKind))
	}

	stats, err := env.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if stats.OutboxPending != 1 || stats.OutboxOldest.IsZero() || stats.Started {
		t.Fatalf("stats %+v", stats)
	}
}

type bareBus struct{}

func (bareBus) Publish(context.Context, ...bus.Envelope) error { return nil }
func (bareBus) Subscribe(context.Context, string, []string, bus.Handler) (bus.Subscription, error) {
	return nopSub{}, nil
}
func (bareBus) Close() error { return nil }

type nopSub struct{}

func (nopSub) Close() error { return nil }
