package ee_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/mem"
	"github.com/clubpay/ronykit/ee/store"
)

func TestActionCtxAPI(t *testing.T) {
	f := newFaultFixture(t, mem.NewStore(), mem.NewBus(mem.RetryBackoff(time.Millisecond)))
	ctx := t.Context()

	_ = f.items.Create(ctx, Item{ID: "i1", OrderID: "o1"})
	_ = f.receipts.Action(func(c *ee.ActionCtx[Receipt, ReceiptRequested]) error {
		return c.Create(&Receipt{ID: "r-" + c.Event().OrderID, OrderID: c.Event().OrderID})
	})

	done := make(chan error, 1)
	_ = f.orders.Action(func(c *ee.ActionCtx[Order, ItemAdded]) error {
		err := func() error {
			if c.Tx() == nil {
				t.Error("Tx() must not be nil")
			}

			// Meta carries the whole envelope.
			m := c.Meta()
			if m.ID != "meta-1" || m.Event != "item.added" || m.PartitionKey != "o1" {
				t.Errorf("meta = %+v", m)
			}
			if m.Headers["trace"] != "t1" || m.Labels["sku"] != "x" {
				t.Errorf("meta headers/labels = %+v", m)
			}

			// Member states are reachable.
			if _, err := c.States[Item]().Get("i1"); err != nil {
				return err
			}

			// Emit goes to the outbox.
			if err := c.Emit(ReceiptRequested{OrderID: "o1"}); err != nil {
				return err
			}

			// A bad event surfaces its encode error.
			emitErr := c.Emit(badEvent{})
			if emitErr == nil || !strings.Contains(emitErr.Error(), "encode bad.evt") {
				t.Errorf("emit bad event: %v", emitErr)
			}

			return nil
		}()
		done <- err

		return err
	})
	f.start(t)

	payload, _ := json.Marshal(ItemAdded{OrderID: "o1", ItemID: "i1"})
	_ = f.bus.Publish(ctx, bus.Envelope{
		ID: "meta-1", Event: "item.added", PartitionKey: "o1", Payload: payload,
		Labels:  map[string]string{"sku": "x"},
		Headers: map[string]string{"trace": "t1"},
	})
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	eventually(t, "the emitted receipt", func() bool {
		_, err := f.receipts.Get(ctx, "r-o1")

		return err == nil
	})
}

func TestActionCtxRequiresKeyedEvent(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	_ = f.orders.Create(ctx, Order{ID: "o1"})

	done := make(chan error, 1)
	_ = f.orders.Action(func(c *ee.ActionCtx[Order, plainNoted]) error {
		done <- func() error {
			_, err := c.State()

			return err
		}()

		return nil
	})
	f.start(t)

	_ = f.env.Send(ctx, plainNoted{Text: "hi"})
	if err := <-done; !errors.Is(err, ee.ErrNoKey) {
		t.Errorf("want ErrNoKey, got %v", err)
	}
}

func TestUndecodableEventIsDropped(t *testing.T) {
	buf := &syncBuffer{}
	bs := mem.NewBus(mem.RetryBackoff(time.Millisecond))
	env := ee.MustNew(ee.WithStore(mem.NewStore()), ee.WithBus(bs),
		ee.WithOutboxPoll(5*time.Millisecond),
		ee.WithLogger(slog.New(slog.NewTextHandler(buf, nil))))
	orders := must(ee.NewStateController[Order](env))
	t.Cleanup(func() { _ = env.Close(); _ = bs.Close() })

	var mu sync.Mutex

	attempts := 0

	_ = orders.Action(func(*ee.ActionCtx[Order, OrderTouched]) error {
		mu.Lock()
		attempts++
		mu.Unlock()

		return nil
	})
	if err := env.Start(t.Context()); err != nil {
		t.Fatal(err)
	}

	// A payload no codec can decode is permanent: log it, ack it, move on.
	_ = bs.Publish(t.Context(), bus.Envelope{
		ID: "bad-1", Event: "order.touched", PartitionKey: "o1", Payload: []byte("{{oops"),
	})

	eventually(t, "the drop log", func() bool {
		return strings.Contains(buf.String(), "dropping event")
	})

	if !strings.Contains(buf.String(), "decode order.touched") {
		t.Errorf("log misses the decode error: %s", buf.String())
	}

	time.Sleep(30 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if attempts != 0 {
		t.Errorf("the handler ran %d times for an undecodable event", attempts)
	}
	if n := strings.Count(buf.String(), "dropping event"); n != 1 {
		t.Errorf("dropped %d times (no redelivery expected), log: %s", n, buf.String())
	}
}

// syncBuffer is an append-only buffer for capture by slog.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.buf.String()
}

func TestDedupMarkerFlow(t *testing.T) {
	t.Run("failing handler leaves no marker", func(t *testing.T) {
		f := newFaultFixture(t, mem.NewStore(), mem.NewBus(mem.RetryBackoff(time.Millisecond)),
			ee.WithMiddleware(ee.Dedup))
		ctx := t.Context()

		_ = f.orders.Create(ctx, Order{ID: "o1"})

		var mu sync.Mutex

		n := 0

		_ = f.orders.Action(func(c *ee.ActionCtx[Order, OrderTouched]) error {
			mu.Lock()
			n++
			first := n == 1
			mu.Unlock()

			if first {
				return errStop // the first attempt fails; no marker must be written
			}

			o, err := c.State()
			if err != nil {
				return err
			}

			o.Amount++
			if err := c.Save(o); err != nil {
				return err
			}

			return nil
		})
		f.start(t)

		_ = f.env.Send(ctx, OrderTouched{OrderID: "o1"})
		eventually(t, "the handler to succeed", func() bool {
			o, _ := f.orders.Get(ctx, "o1")

			return o.Amount == 1
		})

		if k := dedupMarkers(f.store, ctx); k != 1 {
			t.Errorf("dedup markers = %d, want 1", k)
		}
	})

	t.Run("dedup get fault is retried through", func(t *testing.T) {
		faults := newFaultSet()
		f := newFaultFixture(t,
			newFaultStore(func(st *faultStore) { st.txOp = faults.hook() }),
			mem.NewBus(mem.RetryBackoff(time.Millisecond)),
			ee.WithMiddleware(ee.Dedup), ee.WithMaxRetries(4))
		ctx := t.Context()

		_ = f.orders.Create(ctx, Order{ID: "o1"})

		var n int

		_ = f.orders.Action(func(c *ee.ActionCtx[Order, OrderTouched]) error {
			o, err := c.State()
			if err != nil {
				return err
			}

			n++

			o.Amount++
			if err := c.Save(o); err != nil {
				return err
			}

			return nil
		})
		f.start(t)

		faults.armAny("get", ee.DedupKind)
		_ = f.env.Send(ctx, OrderTouched{OrderID: "o1"})

		eventually(t, "one handled delivery", func() bool {
			o, _ := f.orders.Get(ctx, "o1")

			return o.Amount == 1
		})
		if n != 1 {
			t.Errorf("handler ran %d times, want 1", n)
		}
	})
}

func dedupMarkers(st *mem.Store, ctx context.Context) int {
	_ = ctx

	return st.Len(ee.DedupKind)
}

// TestDedupLoserRollsBackAndSkips simulates two replicas racing over the same
// duplicate: the loser loses the marker insert, its unit of work is retried,
// and the retry then sees the winner's marker and skips. Nothing it did (state
// writes, outbox entries) survives.
func TestDedupLoserRollsBackAndSkips(t *testing.T) {
	raced := false

	st := newFaultStore(func(s *faultStore) {
		s.txOp = func(op, kind, id string) error {
			if op == "insert" && kind == ee.DedupKind && id == "state:order/race-1" && !raced {
				raced = true
				// The winning replica commits its marker first.
				_ = s.Store.Tx(context.Background(), func(_ context.Context, tx store.Tx) error {
					return tx.Insert(context.Background(), store.Record{
						Kind: ee.DedupKind, ID: id, Version: 1, Data: []byte(`"winner"`),
					})
				})

				return store.ErrConflict
			}

			return nil
		}
	})
	bs := mem.NewBus(mem.RetryBackoff(time.Millisecond))
	env := ee.MustNew(ee.WithStore(st), ee.WithBus(bs), ee.WithMiddleware(ee.Dedup))
	orders := must(ee.NewStateController[Order](env))
	t.Cleanup(func() { _ = env.Close(); _ = bs.Close() })

	ctx := t.Context()

	_ = orders.Create(ctx, Order{ID: "o1"})

	var mu sync.Mutex

	n := 0

	_ = orders.Action(func(c *ee.ActionCtx[Order, OrderTouched]) error {
		o, err := c.State()
		if err != nil {
			return err
		}

		mu.Lock()
		n++
		mu.Unlock()

		o.Amount++
		if err := c.Save(o); err != nil {
			return err
		}

		return c.Emit(ReceiptRequested{OrderID: o.ID})
	})
	if err := env.Start(ctx); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(OrderTouched{OrderID: "o1"})
	_ = bs.Publish(ctx, bus.Envelope{ID: "race-1", Event: "order.touched", PartitionKey: "o1", Payload: payload})

	// The unit is retried once and immediately skipped; wait for it to finish.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ran := n
		mu.Unlock()

		if k := dedupMarkers(st.Store, ctx); ran == 1 && k == 1 {
			break
		}

		time.Sleep(2 * time.Millisecond)
	}
	if k := dedupMarkers(st.Store, ctx); k != 1 {
		t.Fatalf("dedup markers = %d, want the winner only", k)
	}

	mu.Lock()
	ran := n
	mu.Unlock()
	if ran != 1 {
		t.Errorf("handler ran %d times, want 1", ran)
	}

	// The loser's write was rolled back and its emit never reached the outbox.
	if o, err := orders.Get(ctx, "o1"); err != nil || o.Amount != 0 {
		t.Errorf("loser's write survived: %+v, %v", o, err)
	}
	receipts := receiptIDs(st.Store, ctx)
	if len(receipts) != 0 {
		t.Errorf("the loser's emit leaked: %v", receipts)
	}
}

func receiptIDs(st *mem.Store, ctx context.Context) []string {
	var out []string

	_ = st.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
		for r, err := range tx.Find(ctx, store.Query{Kind: "receipt"}) {
			if err != nil {
				return err
			}

			out = append(out, r.ID)
		}

		return nil
	})

	return out
}

func TestMiddlewareErrorRedelivers(t *testing.T) {
	var (
		mu sync.Mutex

		n int
	)

	mw := func(next ee.Next) ee.Next {
		return func(ctx context.Context, d *ee.Delivery) error {
			if d.Owner != "state:order" || d.Event != "order.touched" || d.Tx == nil || d.ID == "" {
				t.Errorf("delivery = %+v", d)
			}

			mu.Lock()
			n++
			attempt := n
			mu.Unlock()

			if attempt == 1 {
				return errStop // one redelivery expected
			}

			return next(ctx, d)
		}
	}

	bs := mem.NewBus(mem.RetryBackoff(time.Millisecond))
	env := ee.MustNew(ee.WithStore(mem.NewStore()), ee.WithBus(bs), ee.WithMiddleware(mw), ee.WithMaxRetries(4))
	orders := must(ee.NewStateController[Order](env))
	t.Cleanup(func() { _ = env.Close(); _ = bs.Close() })

	ctx := t.Context()

	_ = orders.Create(ctx, Order{ID: "o1"})

	_ = orders.Action(func(c *ee.ActionCtx[Order, OrderTouched]) error {
		o, err := c.State()
		if err != nil {
			return err
		}

		o.Amount++

		return c.Save(o)
	})
	if err := env.Start(ctx); err != nil {
		t.Fatal(err)
	}

	_ = env.Send(ctx, OrderTouched{OrderID: "o1"})

	eventually(t, "the handled event", func() bool {
		o, _ := orders.Get(ctx, "o1")

		return o.Amount == 1
	})

	mu.Lock()
	defer mu.Unlock()

	if n != 2 {
		t.Errorf("middleware ran %d times, want 2 (one redelivery)", n)
	}
}
