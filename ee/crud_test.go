package ee_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/mem"
	"github.com/clubpay/ronykit/ee/store"
)

var errStop = errors.New("stop")

// ghost is never registered; repos for it must fail with ErrNotRegistered.
type ghost struct {
	ID string `ee:"id"`
}

func (ghost) EntityName() string { return "ghost" }

func TestEntityCRUDLifecycle(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	if err := f.orders.Create(ctx, Order{ID: "o1", Merchant: "m1", Amount: 1}); err != nil {
		t.Fatal(err)
	}

	o, err := f.orders.Get(ctx, "o1")
	if err != nil || o.Merchant != "m1" || o.Amount != 1 {
		t.Fatalf("get: %+v, %v", o, err)
	}

	if err := f.orders.Update(ctx, "o1", func(o *Order) error { return errStop }); !errors.Is(err, errStop) {
		t.Errorf("a failing update fn must resurface: %v", err)
	}
	if o, _ := f.orders.Get(ctx, "o1"); o.Amount != 1 {
		t.Errorf("a failing update was committed: %+v", o)
	}

	if err := f.orders.Update(ctx, "o1", func(o *Order) error { o.Amount = 3; return nil }); err != nil {
		t.Fatal(err)
	}
	if o, _ := f.orders.Get(ctx, "o1"); o.Amount != 3 {
		t.Errorf("update was not applied: %+v", o)
	}

	if _, err := f.orders.Get(ctx, "missing"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("get of a missing state: %v", err)
	}

	if err := f.orders.Update(ctx, "missing", func(*Order) error { return nil }); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("update of a missing state: %v", err)
	}

	if err := f.orders.Delete(ctx, "o1"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.orders.Get(ctx, "o1"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("after delete: %v", err)
	}
	if err := f.orders.Delete(ctx, "o1"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("second delete: %v", err)
	}
}

func TestActionUnitSemantics(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	_ = f.orders.Create(ctx, Order{ID: "o1"})
	_ = f.items.Create(ctx, Item{ID: "i1", OrderID: "o1"})

	done := make(chan error, 1)
	_ = f.orders.Action(func(c *ee.ActionCtx[Order, OrderTouched]) error {
		err := func() error {
			if c.Tx() == nil {
				t.Error("Tx() must not be nil")
			}

			// The identity map: one id, one pointer, within the unit.
			a, err := c.Get("o1")
			if err != nil {
				return err
			}
			b, err := c.Get("o1")
			if err != nil {
				return err
			}
			if a != b {
				t.Error("Get must return the same pointer for the same id")
			}

			// save -> read -> save within the same unit.
			a.Amount = 7
			if err := c.Save(a); err != nil {
				return err
			}
			if c2, err := c.Get("o1"); err != nil || c2.Amount != 7 {
				t.Errorf("saved value not readable: %+v, %v", c2, err)
			}

			a.Amount = 8
			if err := c.Save(a); err != nil {
				return err
			}

			// Find returns the tracked state (identity map) and honors an early
			// break from the consumer.
			it, err := c.Entities[Item]().Get("i1")
			if err != nil {
				return err
			}

			n := 0
			for got, err := range c.Entities[Item]().Find(ee.Query{}) {
				if err != nil {
					return err
				}
				if got != it {
					t.Errorf("Find must return the tracked pointer, got %+v", got)
				}
				n++
				break
			}
			if n != 1 {
				t.Errorf("find visited %d rows", n)
			}

			if err := c.Entities[Item]().Delete(it); err != nil {
				return err
			}

			// A tracked-and-deleted state is gone for the rest of the unit.
			if err := c.Entities[Item]().Save(it); !errors.Is(err, ee.ErrNotFound) {
				t.Errorf("save of a deleted state: %v", err)
			}
			if _, err := c.Entities[Item]().Get("i1"); !errors.Is(err, ee.ErrNotFound) {
				t.Errorf("get of a deleted state: %v", err)
			}
			n = 0
			for _, err := range c.Entities[Item]().Find(ee.Query{}) {
				if err != nil {
					return err
				}
				n++
			}
			if n != 0 {
				t.Errorf("Find must skip states deleted in the unit, got %d", n)
			}

			if err := c.Delete(&Order{ID: "zz"}); !errors.Is(err, ee.ErrNotLoaded) {
				t.Errorf("delete of an untracked state: %v", err)
			}

			// Creating a duplicate inside the unit conflicts.
			if err := c.Create(&Order{ID: "o1"}); !errors.Is(err, ee.ErrConflict) {
				t.Errorf("duplicate create: %v", err)
			}
			if err := c.Create(&Order{}); err == nil || !strings.Contains(err.Error(), "empty id") {
				t.Errorf("create with an empty id: %v", err)
			}

			// Repos for states of no scope fail per operation.
			g := c.Entities[ghost]()
			if _, err := g.Get("x"); !errors.Is(err, ee.ErrNotRegistered) {
				t.Errorf("unregistered get: %v", err)
			}
			for _, err := range g.Find(ee.Query{}) {
				if !errors.Is(err, ee.ErrNotRegistered) {
					t.Errorf("unregistered find: %v", err)
				}
			}
			if err := g.Create(&ghost{ID: "x"}); !errors.Is(err, ee.ErrNotRegistered) {
				t.Errorf("unregistered create: %v", err)
			}
			if err := g.Save(&ghost{ID: "x"}); !errors.Is(err, ee.ErrNotRegistered) {
				t.Errorf("unregistered save: %v", err)
			}
			if err := g.Delete(&ghost{ID: "x"}); !errors.Is(err, ee.ErrNotRegistered) {
				t.Errorf("unregistered delete: %v", err)
			}

			return nil
		}()
		done <- err

		return err
	})
	f.start(t)

	_ = f.env.Send(ctx, OrderTouched{OrderID: "o1"})
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	// done fires before the unit commits; wait for the commit to land.
	eventually(t, "final value", func() bool {
		o, _ := f.orders.Get(ctx, "o1")

		return o.Amount == 8
	})

	// The item stayed deleted; the order id zz was never created.
	if _, err := f.items.Get(ctx, "i1"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("item resurrection: %v", err)
	}
	if _, err := f.orders.Get(ctx, "zz"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("untracked create leaked: %v", err)
	}
}

func TestEntityFindStopAndErrors(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	for i := range 3 {
		_ = f.orders.Create(ctx, Order{ID: "o" + fmt.Sprint(i)})
	}

	// An early break stops the stream without a trailing zero value or error.
	type row struct {
		o   Order
		err error
	}

	var rows []row
	for o, err := range f.orders.Find(ctx, ee.Query{}) {
		rows = append(rows, row{o, err})
		break
	}
	if len(rows) != 1 || rows[0].err != nil || rows[0].o.ID == "" {
		t.Errorf("early break: %+v", rows)
	}
}

func TestAutoSave(t *testing.T) {
	f := newFixture(t, ee.WithAutoSave())
	ctx := t.Context()

	_ = f.orders.Create(ctx, Order{ID: "o1"})
	_ = f.orders.Create(ctx, Order{ID: "o2", Amount: 1})
	_ = f.items.Create(ctx, Item{ID: "i1", OrderID: "o1"})

	done := make(chan error, 1)
	_ = f.orders.Action(func(c *ee.ActionCtx[Order, OrderTouched]) error {
		err := func() error {
			it, err := c.Entities[Item]().Get("i1")
			if err != nil {
				return err
			}
			if err := c.Entities[Item]().Delete(it); err != nil {
				return err
			}

			o, err := c.Get("o1")
			if err != nil {
				return err
			}
			o.Amount = 5 // no explicit Save; commit must save the dirty one

			return nil
		}()
		done <- err

		return err
	})
	f.start(t)

	_ = f.env.Send(ctx, OrderTouched{OrderID: "o1"})
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	eventually(t, "autosave", func() bool {
		o, _ := f.orders.Get(ctx, "o1")

		return o.Amount == 5
	})
	if _, err := f.items.Get(ctx, "i1"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("auto-save resurrected a deleted state: %v", err)
	}

	// o2 was never touched: its record must stay at version 1.
	ver := storeVersion(f.store, ctx, "order", "o2")
	if ver != 1 {
		t.Errorf("untouched state was rewritten, version = %d", ver)
	}
}

func storeVersion(st *mem.Store, ctx context.Context, kind, id string) uint64 {
	var v uint64

	_ = st.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
		r, err := tx.Get(ctx, kind, id)
		if err == nil {
			v = r.Version
		}

		return err
	})

	return v
}

func TestPanicIsRecovered(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	_ = f.orders.Create(ctx, Order{ID: "o1"})

	err := f.orders.Update(ctx, "o1", func(*Order) error { panic("boom") })
	if err == nil || !strings.Contains(err.Error(), "panic: boom") {
		t.Fatalf("want the recovered panic, got %v", err)
	}

	if o, _ := f.orders.Get(ctx, "o1"); o.Amount != 0 {
		t.Errorf("a panicking unit was committed: %+v", o)
	}
}

func TestFaultIsRetriedThrough(t *testing.T) {
	faults := newFaultSet()
	f := newFaultFixture(t, newFaultStore(func(st *faultStore) { st.txOp = faults.hook() }),
		mem.NewBus(mem.RetryBackoff(time.Millisecond)), ee.WithMaxRetries(4))
	ctx := t.Context()

	_ = f.orders.Create(ctx, Order{ID: "o1"})
	_ = f.items.Create(ctx, Item{ID: "i1"})

	// One faulted Update is retried and commits.
	faults.arm("update", "order", "o1")
	if err := f.orders.Update(ctx, "o1", func(o *Order) error { o.Amount = 9; return nil }); err != nil {
		t.Fatal(err)
	}
	if o, _ := f.orders.Get(ctx, "o1"); o.Amount != 9 {
		t.Errorf("faulted update lost: %+v", o)
	}

	// One faulted Delete is retried and commits.
	faults.arm("delete", "item", "i1")
	if err := f.items.Delete(ctx, "i1"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.items.Get(ctx, "i1"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("faulted delete lost: %v", err)
	}
}

func TestCreateWithoutRetry(t *testing.T) {
	faults := newFaultSet()
	f := newFaultFixture(t, newFaultStore(func(st *faultStore) { st.txOp = faults.hook() }),
		mem.NewBus(mem.RetryBackoff(time.Millisecond)))
	ctx := t.Context()

	faults.arm("insert", "order", "o1")
	if err := f.orders.Create(ctx, Order{ID: "o1"}); !errors.Is(err, ee.ErrConflict) {
		t.Fatalf("a faulted create must surface, got %v", err)
	}

	// The store stays empty.
	if _, err := f.orders.Get(ctx, "o1"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("a faulted create committed: %v", err)
	}
}

// ---- poison codec ---------------------------------------------------------
// poisonCodec is JSON plus one value it can decode but never re-encode.

type evilData struct{ N int }

type poisonOrder struct {
	ID   string `ee:"id"`
	Evil *evilData
}

func (poisonOrder) EntityName() string { return "poison" }

type poisonTouched struct {
	ID string
}

func (poisonTouched) EventName() string { return "poison.touched" }
func (e poisonTouched) Key() string     { return e.ID }

func TestCorruptRecordDecode(t *testing.T) {
	f, poison := newPoisonFixture(t)
	ctx := t.Context()

	// Seed records bypassing ee: one not decodable at all, one decodable but
	// never re-encodable by the codec, one ordinary.
	seeds := map[string]string{
		"undecodable": "not json at all",
		"evil-data":   `{"Evil":{"N":1}}`,
		"plain":       `{"ID":"plain"}`,
	}
	_ = f.store.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
		for id, data := range seeds {
			if err := tx.Insert(ctx, store.Record{Kind: "poison", ID: id, Version: 1, Data: []byte(data)}); err != nil {
				return err
			}
		}

		return nil
	})

	if _, err := poison.Get(ctx, "undecodable"); err == nil || !strings.Contains(err.Error(), "decode poison/undecodable") {
		t.Errorf("undecodable record: %v", err)
	}

	// Get of a decodable record still fails at the snapshot re-encode.
	if _, err := poison.Get(ctx, "evil-data"); err == nil || !strings.Contains(err.Error(), "encode poison/evil-data") {
		t.Errorf("re-encode failure: %v", err)
	}

	// Find fails per record: the undecodable one at the store decoder, the
	// evil one at the codec, and both stop the stream with an error.
	for _, err := range poison.Find(ctx, ee.Query{}) {
		if err == nil {
			t.Errorf("find over corrupt records delivered a value")

			break
		}

		break
	}

	var got error
	for v, err := range poison.Find(ctx, ee.Query{}) {
		if err == nil && v.ID == "plain" {
			t.Errorf("corrupt records must stop the stream before plain")

			continue
		}

		got = err

		break
	}
	if got == nil {
		t.Error("find over corrupt records must yield an error")
	}
}

func TestPoisonRefusesToEncode(t *testing.T) {
	_, poison := newPoisonFixture(t)
	ctx := t.Context()

	if err := poison.Create(ctx, poisonOrder{ID: "plain", Evil: &evilData{}}); err == nil ||
		!strings.Contains(err.Error(), "encode poison/plain") {
		t.Errorf("create of a poison value: %v", err)
	}

	_ = poison.Create(ctx, poisonOrder{ID: "plain"})
	if err := poison.Update(ctx, "plain", func(p *poisonOrder) error { p.Evil = &evilData{}; return nil }); err == nil ||
		!strings.Contains(err.Error(), "encode poison/plain") {
		t.Errorf("save of a poison mutation: %v", err)
	}

	// The failed save changed nothing.
	if p, err := poison.Get(ctx, "plain"); err != nil || p.Evil != nil {
		t.Errorf("poison save was committed: %+v, %v", p, err)
	}
}

func TestAutoSavePoisonFailsAtCommit(t *testing.T) {
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
	env := ee.MustNew(ee.WithStore(mem.NewStore()), ee.WithBus(bs), ee.WithCodec(newPoisonCodec()),
		ee.WithAutoSave(), ee.WithOutboxPoll(5*time.Millisecond),
		ee.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	poison := must(ee.NewEntityController[poisonOrder](env))
	t.Cleanup(func() { _ = env.Close(); _ = bs.Close() })

	ctx := t.Context()
	_ = poison.Create(ctx, poisonOrder{ID: "p1"})

	handled := make(chan struct{}, 3)
	_ = poison.Action(func(c *ee.ActionCtx[poisonOrder, poisonTouched]) error {
		o, err := c.Entity()
		if err != nil {
			return err
		}

		o.Evil = &evilData{} // no Save: the commit-time save must fail
		handled <- struct{}{}

		return nil
	})
	if err := env.Start(ctx); err != nil {
		t.Fatal(err)
	}

	_ = env.Send(ctx, poisonTouched{ID: "p1"})
	<-handled

	eventually(t, "the poison dead letter", func() bool {
		mu.Lock()
		defer mu.Unlock()

		return slices.Equal(dead, []string{"poison.touched"})
	})

	mu.Lock()
	defer mu.Unlock()

	if !slices.Equal(dead, []string{"poison.touched"}) {
		t.Errorf("dead letters = %v, want the poison event", dead)
	}

	if p, err := poison.Get(ctx, "p1"); err != nil || p.Evil != nil {
		t.Errorf("a poison auto-save was committed: %+v, %v", p, err)
	}
}

// TestAutoSaveFaultRetriedThrough: a faulted commit-time save (WithAutoSave)
// fails the unit and the retry applies it once the fault clears.
func TestAutoSaveFaultRetriedThrough(t *testing.T) {
	faults := newFaultSet()
	f := newFaultFixture(t,
		newFaultStore(func(st *faultStore) { st.txOp = faults.hook() }),
		mem.NewBus(mem.RetryBackoff(time.Millisecond)),
		ee.WithAutoSave(), ee.WithMaxRetries(4))
	ctx := t.Context()

	_ = f.orders.Create(ctx, Order{ID: "o1"})

	_ = f.orders.Action(func(c *ee.ActionCtx[Order, OrderTouched]) error {
		o, err := c.Entity()
		if err != nil {
			return err
		}

		o.Amount++ // no explicit Save

		return nil
	})
	f.start(t)

	faults.arm("update", "order", "o1")
	_ = f.env.Send(ctx, OrderTouched{OrderID: "o1"})

	eventually(t, "the autosaved value", func() bool {
		o, _ := f.orders.Get(ctx, "o1")

		return o.Amount == 1
	})
}
