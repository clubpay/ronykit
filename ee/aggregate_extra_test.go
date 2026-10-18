package ee_test

import (
	"context"
	"encoding/json"
	"errors"
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

// valueAgg exercises a value-struct root, a pointer slice with nil entries and
// a pointer map with nil values. With no `ee:"root"` tag, the first
// single-state member is the root.
type valueAgg struct {
	Order Order
	Extra []*Item
	Ptrs  map[string]*Payment
}

func (valueAgg) AggregateName() string { return "value-agg" }

func (a *valueAgg) Load(ctx *ee.LoadCtx, id string) error {
	o, err := ctx.Entities[Order]().Get(id)
	if err != nil {
		return err
	}

	a.Order = *o

	return nil
}

type nilRootAgg struct {
	Order *Order
}

func (nilRootAgg) AggregateName() string { return "nil-root-agg" }

func (*nilRootAgg) Load(*ee.LoadCtx, string) error { return nil } // leaves the root nil

type loadErrAgg struct {
	Order *Order
}

func (loadErrAgg) AggregateName() string { return "load-err-agg" }

func (*loadErrAgg) Load(*ee.LoadCtx, string) error { return errStop }

// saverAgg replaces the default diff-and-save.
type saverAgg struct {
	Order *Order
	Items []Item
}

func (saverAgg) AggregateName() string { return "saver-agg" }

func (a *saverAgg) Load(ctx *ee.LoadCtx, id string) error {
	o, err := ctx.Entities[Order]().Get(id)
	if err != nil {
		return err
	}

	a.Order = o

	return nil
}

func (a *saverAgg) Save(ctx *ee.SaveCtx) error {
	if ctx.Tx() == nil {
		return errNoSaveTx
	}

	a.Order.Amount += 100

	// The root is tracked when the aggregate was loaded; on Create it is new.
	if err := ctx.Entities[Order]().Save(a.Order); err != nil {
		if !errors.Is(err, ee.ErrNotLoaded) {
			return err
		}

		return ctx.Entities[Order]().Create(a.Order)
	}

	return nil
}

// phantomAgg reaches one member through the backend escape hatch instead of
// the repo, so it is never tracked.
type phantomAgg struct {
	Order *Order
	Items []Item
}

func (phantomAgg) AggregateName() string { return "phantom-agg" }

func (a *phantomAgg) Load(ctx *ee.LoadCtx, id string) error {
	o, err := ctx.Entities[Order]().Get(id)
	if err != nil {
		return err
	}

	a.Order = o

	_, err = ctx.Tx().Get(ctx, "item", "phantom-i")
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}

	a.Items = append(a.Items, Item{ID: "phantom-i", OrderID: id})

	return nil
}

func TestAggregateValueShapes(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	values := must(ee.NewAggregateController[valueAgg](f.env))

	err := values.Create(ctx, &valueAgg{
		Order: Order{ID: "v1", Amount: 1},
		Extra: []*Item{nil, {ID: "x1", OrderID: "v1"}},
		Ptrs:  map[string]*Payment{"cash": nil},
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := f.items.Get(ctx, "x1"); err != nil {
		t.Errorf("pointer-slice member not created: %v", err)
	}

	a, err := values.Get(ctx, "v1")
	if err != nil || a.Order.ID != "v1" {
		t.Fatalf("get: %+v, %v", a, err)
	}

	if err := values.Update(ctx, "v1", func(a *valueAgg) error {
		a.Order.Amount = 2
		a.Extra = append(a.Extra, &Item{ID: "x2", OrderID: "v1"})
		a.Ptrs = map[string]*Payment{"card": {ID: "p1", OrderID: "v1", Method: "card"}}

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if p, err := f.payments.Get(ctx, "p1"); err != nil || p.Method != "card" {
		t.Errorf("ptr-map member not created: %+v, %v", p, err)
	}

	// Delete removes what Load tracked (the root); a pointer member that Load
	// left nil was never part of the aggregate.
	if err := values.Delete(ctx, "v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.orders.Get(ctx, "v1"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("root not deleted: %v", err)
	}
	if _, err := f.items.Get(ctx, "x1"); err != nil {
		t.Errorf("x1 must survive: Load never loaded it, Delete only removes what it did: %v", err)
	}
}

func TestAggregateLoadAndSaveErrors(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	loadErrs := must(ee.NewAggregateController[loadErrAgg](f.env))
	nilRoots := must(ee.NewAggregateController[nilRootAgg](f.env))

	// Load errors resurface.
	if _, err := loadErrs.Get(ctx, "o1"); !errors.Is(err, errStop) {
		t.Errorf("load error: %v", err)
	}

	// Load that leaves the root nil fails the collect.
	if _, err := nilRoots.Get(ctx, "o1"); err == nil || !strings.Contains(err.Error(), "root Order is nil") {
		t.Errorf("nil root at load: %v", err)
	}
	if err := nilRoots.Update(ctx, "o1", func(*nilRootAgg) error { return nil }); err == nil ||
		!strings.Contains(err.Error(), "root Order is nil") {
		t.Errorf("nil root at load update: %v", err)
	}

	if err := f.aggs.Delete(ctx, "missing"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("delete of a missing aggregate: %v", err)
	}
	if err := f.aggs.Update(ctx, "missing", func(*OrderAgg) error { return nil }); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("update of a missing aggregate: %v", err)
	}

	// An update whose fn fails rolls back.
	_ = f.aggs.Create(ctx, &OrderAgg{Order: &Order{ID: "o1"}})
	if err := f.aggs.Update(ctx, "o1", func(*OrderAgg) error { return errStop }); !errors.Is(err, errStop) {
		t.Errorf("fn error: %v", err)
	}
	if o, _ := f.orders.Get(ctx, "o1"); o.Amount != 0 {
		t.Errorf("failed fn was committed: %+v", o)
	}

	// Duplicate member ids conflict.
	err := f.aggs.Create(ctx, &OrderAgg{
		Order: &Order{ID: "o2"},
		Items: []Item{{ID: "dup", OrderID: "o2"}, {ID: "dup", OrderID: "o2"}},
	})
	if err == nil || !strings.Contains(err.Error(), "item/dup twice") {
		t.Errorf("duplicate member: %v", err)
	}

	// A member with an empty id is refused at collect time.
	err = f.aggs.Create(ctx, &OrderAgg{Order: &Order{}})
	if err == nil || !strings.Contains(err.Error(), "empty id") {
		t.Errorf("empty-id member: %v", err)
	}
}

func TestAggregateNilRootAtSave(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	_ = f.aggs.Create(ctx, &OrderAgg{Order: &Order{ID: "o1"}})

	err := f.aggs.Update(ctx, "o1", func(a *OrderAgg) error { a.Order = nil; return nil })
	if err == nil || !strings.Contains(err.Error(), "root Order is nil") {
		t.Errorf("nil root at save: %v", err)
	}

	// Nothing was written.
	if _, err := f.payments.Get(ctx, "?"); err != nil && !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("view broken: %v", err)
	}
}

func TestAggregateMemberVersions(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	_ = f.aggs.Create(ctx, &OrderAgg{
		Order:    &Order{ID: "o1", Amount: 1},
		Items:    []Item{{ID: "i1", OrderID: "o1"}},
		Payments: map[string]Payment{"card": {ID: "p1", OrderID: "o1", Method: "card"}},
	})

	// Touching only the root leaves unchanged members alone but always bumps
	// the root version.
	if err := f.aggs.Update(ctx, "o1", func(a *OrderAgg) error {
		a.Order.Amount = 2

		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if v := storeVersion(f.store, ctx, "item", "i1"); v != 1 {
		t.Errorf("unchanged member rewritten, version = %d", v)
	}
	if v := storeVersion(f.store, ctx, "order", "o1"); v != 2 {
		t.Errorf("root not rewritten, version = %d", v)
	}

	// A changed member is rewritten.
	if err := f.aggs.Update(ctx, "o1", func(a *OrderAgg) error {
		a.Items[0].SKU = "sku-1"

		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if v := storeVersion(f.store, ctx, "item", "i1"); v != 2 {
		t.Errorf("changed member not rewritten, version = %d", v)
	}
}

func TestAggregateSaver(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	savers, err := ee.NewAggregateController[saverAgg](f.env)
	if err != nil {
		t.Fatal(err)
	}

	err = savers.Create(ctx, &saverAgg{Order: &Order{ID: "s1"}, Items: []Item{{ID: "never", OrderID: "s1"}}})
	if err != nil {
		t.Fatal(err)
	}

	// A Saver replaces the diff-and-save: the item was never written.
	if _, err := f.items.Get(ctx, "never"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("saver wrote members: %v", err)
	}
	if o, err := f.orders.Get(ctx, "s1"); err != nil || o.Amount != 100 {
		t.Errorf("saver did not run: %+v, %v", o, err)
	}

	if err := savers.Update(ctx, "s1", func(a *saverAgg) error {
		a.Items = append(a.Items, Item{ID: "never2", OrderID: "s1"})

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if o, err := f.orders.Get(ctx, "s1"); err != nil || o.Amount != 200 {
		t.Errorf("saver update: %+v, %v", o, err)
	}
	if _, err := f.items.Get(ctx, "never2"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("saver wrote members: %v", err)
	}
}

func TestAggregateDeleteSkipsUntracked(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	// The phantom member exists in the store but is reached through Tx().
	_ = f.store.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
		data, _ := json.Marshal(Item{ID: "phantom-i", OrderID: "px"})

		return tx.Insert(ctx, store.Record{Kind: "item", ID: "phantom-i", Version: 1, Data: data})
	})

	_ = f.orders.Create(ctx, Order{ID: "px"})

	phantoms, err := ee.NewAggregateController[phantomAgg](f.env)
	if err != nil {
		t.Fatal(err)
	}

	if err := phantoms.Delete(ctx, "px"); err != nil {
		t.Fatal(err)
	}

	if _, err := f.orders.Get(ctx, "px"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("root not deleted: %v", err)
	}
	// The untracked member is left alone: deleting requires a tracked version.
	if v := storeVersion(f.store, ctx, "item", "phantom-i"); v != 1 {
		t.Errorf("untracked member was deleted, version = %d", v)
	}
}

func TestAggregateFaultsRetriedThrough(t *testing.T) {
	faults := newFaultSet()
	st := newFaultStore(func(s *faultStore) { s.txOp = faults.hook() })
	f := newFaultFixture(t, st, mem.NewBus(mem.RetryBackoff(time.Millisecond)), ee.WithMaxRetries(4))
	ctx := t.Context()

	_ = f.aggs.Create(ctx, &OrderAgg{
		Order: &Order{ID: "o1"},
		Items: []Item{{ID: "i1", OrderID: "o1"}},
	})

	// A faulted member save is retried.
	faults.arm("update", "order", "o1")
	if err := f.aggs.Update(ctx, "o1", func(a *OrderAgg) error {
		a.Order.Amount = 7
		a.Items[0].SKU = "s"

		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if o, _ := f.orders.Get(ctx, "o1"); o.Amount != 7 {
		t.Errorf("faulted root save lost: %+v", o)
	}

	// A faulted member delete is retried.
	faults.arm("delete", "item", "i1")
	if err := f.aggs.Update(ctx, "o1", func(a *OrderAgg) error {
		a.Items = nil

		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.items.Get(ctx, "i1"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("faulted member delete lost: %v", err)
	}

	// A faulted delete during Aggregate.Delete is retried as well.
	_ = f.aggs.Create(ctx, &OrderAgg{
		Order:    &Order{ID: "o2"},
		Items:    []Item{{ID: "i2", OrderID: "o2"}},
		Payments: map[string]Payment{"card": {ID: "p2", OrderID: "o2"}},
	})
	faults.arm("delete", "payment", "p2")
	if err := f.aggs.Delete(ctx, "o2"); err != nil {
		t.Fatal(err)
	}
	if v := storeVersion(f.store, ctx, "payment", "p2"); v != 0 {
		t.Errorf("payment survived Delete, version = %d", v)
	}
}

func TestAggregateConflictsSerialize(t *testing.T) {
	f := newFaultFixture(t, mem.NewStore(), mem.NewBus(mem.RetryBackoff(time.Millisecond)),
		ee.WithMaxRetries(100))
	ctx := context.Background()

	_ = f.aggs.Create(ctx, &OrderAgg{Order: &Order{ID: "o1"}})

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			err := f.aggs.Update(ctx, "o1", func(a *OrderAgg) error {
				a.Order.Amount++
				time.Sleep(time.Millisecond) // widen the race window

				return nil
			})
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()

	if a, err := f.aggs.Get(ctx, "o1"); err != nil || a.Order.Amount != 8 {
		t.Errorf("lost updates: %+v, %v", a, err)
	}
}

func TestAggregateActionAPI(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	_ = f.orders.Create(ctx, Order{ID: "o1"})
	_ = f.receipts.Action(func(c *ee.ActionCtx[Receipt, ReceiptRequested]) error {
		return c.Create(&Receipt{ID: "r-" + c.Event().OrderID, OrderID: c.Event().OrderID})
	})

	done := make(chan error, 1)
	_ = f.aggs.Action(func(c *ee.AggregateCtx[OrderAgg, ItemAdded]) error {
		err := func() error {
			m := c.Meta()
			if m.PartitionKey != "o1" || m.Event != "item.added" {
				t.Errorf("meta = %+v", m)
			}

			if c.Tx() == nil {
				t.Error("Tx() must not be nil")
			}

			if _, err := c.Entities[Item]().Get("missing"); !errors.Is(err, ee.ErrNotFound) {
				t.Errorf("states[get missing] = %v", err)
			}

			a1, err := c.Aggregate()
			if err != nil {
				return err
			}
			a2, err := c.Aggregate() // cached
			if err != nil {
				return err
			}
			if a1 != a2 {
				t.Error("Aggregate() must return the same pointer")
			}

			a1.Items = append(a1.Items, Item{ID: "i9", OrderID: "o1", SKU: "late"})
			a1.Order.Amount++

			if err := c.Emit(ReceiptRequested{OrderID: "o1"}); err != nil {
				return err
			}

			return c.Save(a1)
		}()
		done <- err

		return err
	})
	f.start(t)

	_ = f.env.Send(ctx, ItemAdded{OrderID: "o1", ItemID: "i9"})
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	// done fires before the unit commits; wait for the commit to land.
	eventually(t, "committed member and root", func() bool {
		it, itemErr := f.items.Get(ctx, "i9")
		o, orderErr := f.orders.Get(ctx, "o1")

		return itemErr == nil && it.SKU == "late" && orderErr == nil && o.Amount == 1
	})

	eventually(t, "emitted receipt", func() bool {
		_, err := f.receipts.Get(ctx, "r-o1")

		return err == nil
	})
}

func TestAggregateActionAutoSave(t *testing.T) {
	f := newFaultFixture(t, mem.NewStore(), mem.NewBus(mem.RetryBackoff(time.Millisecond)),
		ee.WithAutoSave())
	ctx := t.Context()

	_ = f.aggs.Create(ctx, &OrderAgg{Order: &Order{ID: "o1"}})

	_ = f.aggs.Action(func(c *ee.AggregateCtx[OrderAgg, ItemAdded]) error {
		a, err := c.Aggregate()
		if err != nil {
			return err
		}

		a.Order.Amount++ // no Save: the unit must auto-save it

		return nil
	})
	f.start(t)

	_ = f.env.Send(ctx, ItemAdded{OrderID: "o1"})
	eventually(t, "the auto-saved root", func() bool {
		o, _ := f.orders.Get(ctx, "o1")

		return o.Amount == 1
	})

	// An explicit Save is not repeated by auto-save.
	v := storeVersion(f.store, ctx, "order", "o1")
	if v != 2 {
		t.Errorf("root version = %d, want 2", v)
	}
}

type aggEmptyEvt struct{}

func (aggEmptyEvt) EventName() string { return "" }
func (aggEmptyEvt) Key() string       { return "" }

func TestAggregateActionErrors(t *testing.T) {
	f := newFixture(t)

	if err := f.aggs.Action(func(*ee.AggregateCtx[OrderAgg, aggEmptyEvt]) error { return nil }); !errors.Is(err, ee.ErrInvalidSchema) {
		t.Errorf("empty EventName: %v", err)
	}

	if err := f.orders.Action(func(*ee.ActionCtx[Order, OrderPaid]) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := f.aggs.Action(func(*ee.AggregateCtx[OrderAgg, OrderPaid]) error { return nil }); !errors.Is(err, ee.ErrAlreadyRegistered) {
		t.Errorf("cross-controller claim: %v", err)
	}
}

func TestAggregateActionRollback(t *testing.T) {
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
		ee.WithOutboxPoll(5*time.Millisecond),
		ee.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	orders := must(ee.NewEntityController[Order](env))
	items := must(ee.NewEntityController[Item](env))
	_ = must(ee.NewEntityController[Payment](env))
	aggs := must(ee.NewAggregateController[OrderAgg](env))
	t.Cleanup(func() { _ = env.Close(); _ = bs.Close() })

	ctx := t.Context()

	_ = aggs.Create(ctx, &OrderAgg{Order: &Order{ID: "o1"}})

	_ = aggs.Action(func(c *ee.AggregateCtx[OrderAgg, ItemAdded]) error {
		a, err := c.Aggregate()
		if err != nil {
			return err
		}

		a.Items = append(a.Items, Item{ID: "lost", OrderID: "o1"})
		a.Order.Amount++

		return errStop
	})
	if err := env.Start(ctx); err != nil {
		t.Fatal(err)
	}

	_ = env.Send(ctx, ItemAdded{OrderID: "o1", ItemID: "lost"})

	eventually(t, "the dead letter", func() bool {
		mu.Lock()
		defer mu.Unlock()

		return slices.Equal(dead, []string{"item.added"})
	})

	if o, _ := orders.Get(ctx, "o1"); o.Amount != 0 {
		t.Errorf("failed action was committed: %+v", o)
	}
	if _, err := items.Get(ctx, "lost"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("failed action wrote a member: %v", err)
	}
}

func TestAggregateActionMissingID(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	done := make(chan error, 1)
	_ = f.aggs.Action(func(c *ee.AggregateCtx[OrderAgg, ItemAdded]) error {
		done <- func() error {
			_, err := c.Aggregate()

			return err
		}()

		return nil
	})
	f.start(t)

	_ = f.env.Send(ctx, ItemAdded{OrderID: "ghost"})
	if err := <-done; !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("missing aggregate: %v", err)
	}
}

var errNoSaveTx = errors.New("SaveCtx.Tx() is nil")

func TestAggregateUndecodableEventDropped(t *testing.T) {
	st := mem.NewStore()
	bs := mem.NewBus(mem.RetryBackoff(time.Millisecond), mem.MaxAttempts(2))
	env := ee.MustNew(ee.WithStore(st), ee.WithBus(bs),
		ee.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	_ = must(ee.NewEntityController[Order](env))
	_ = must(ee.NewEntityController[Item](env))
	_ = must(ee.NewEntityController[Payment](env))
	aggs := must(ee.NewAggregateController[OrderAgg](env))
	t.Cleanup(func() { _ = env.Close(); _ = bs.Close() })

	_ = aggs.Action(func(*ee.AggregateCtx[OrderAgg, ItemAdded]) error {
		t.Error("a garbage event reached the aggregate handler")

		return nil
	})
	if err := env.Start(t.Context()); err != nil {
		t.Fatal(err)
	}

	_ = bs.Publish(t.Context(), bus.Envelope{
		ID: "garbage", Event: "item.added", PartitionKey: "o1", Payload: []byte("{{oops"),
	})

	time.Sleep(60 * time.Millisecond) // no redelivery expected
}

func TestAggregateCtxSaveError(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	_ = f.aggs.Create(ctx, &OrderAgg{Order: &Order{ID: "o1"}})

	done := make(chan error, 1)
	_ = f.aggs.Action(func(c *ee.AggregateCtx[OrderAgg, ItemAdded]) error {
		done <- func() error {
			a, err := c.Aggregate()
			if err != nil {
				return err
			}

			a.Order = nil // collect refuses a nil root at save

			return c.Save(a)
		}()

		return nil
	})
	f.start(t)

	_ = f.env.Send(ctx, ItemAdded{OrderID: "o1", ItemID: "i"})
	if err := <-done; err == nil || !strings.Contains(err.Error(), "root Order is nil") {
		t.Errorf("save with a nil root: %v", err)
	}
}
