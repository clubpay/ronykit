package ee_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/mem"
)

type Order struct {
	ID       string `ee:"id"`
	Merchant string
	Amount   int
	Status   string
}

func (Order) EntityName() string { return "order" }

type Receipt struct {
	ID      string `ee:"id"`
	OrderID string
}

func (Receipt) EntityName() string { return "receipt" }

type Item struct {
	ID      string `ee:"id"`
	OrderID string
	SKU     string
}

func (Item) EntityName() string { return "item" }

type Payment struct {
	ID      string `ee:"id"`
	OrderID string
	Method  string
	Amount  int
}

func (Payment) EntityName() string { return "payment" }

type OrderPaid struct {
	OrderID  string
	Merchant string
}

func (OrderPaid) EventName() string { return "order.paid" }
func (e OrderPaid) Key() string     { return e.OrderID }
func (e OrderPaid) Labels() ee.L    { return ee.L{"merchant": e.Merchant} }

type ReceiptRequested struct{ OrderID string }

func (ReceiptRequested) EventName() string { return "receipt.requested" }
func (e ReceiptRequested) Key() string     { return e.OrderID }

type OrderTouched struct{ OrderID string }

func (OrderTouched) EventName() string { return "order.touched" }
func (e OrderTouched) Key() string     { return e.OrderID }

type ItemAdded struct{ OrderID, ItemID, SKU string }

func (ItemAdded) EventName() string { return "item.added" }
func (e ItemAdded) Key() string     { return e.OrderID }

type OrderAgg struct {
	Order    *Order
	Items    []Item
	Payments map[string]Payment
}

func (OrderAgg) AggregateName() string { return "order-agg" }

func (a *OrderAgg) Load(ctx *ee.LoadCtx, id string) error {
	o, err := ctx.Entities[Order]().Get(id)
	if err != nil {
		return err
	}

	a.Order = o

	for it, err := range ctx.Entities[Item]().Find(ee.Where("OrderID", id)) {
		if err != nil {
			return err
		}

		a.Items = append(a.Items, *it)
	}

	a.Payments = map[string]Payment{}

	for p, err := range ctx.Entities[Payment]().Find(ee.Where("OrderID", id)) {
		if err != nil {
			return err
		}

		a.Payments[p.Method] = *p
	}

	return nil
}

type fixture struct {
	env      *ee.Env
	store    *mem.Store
	bus      *mem.Bus
	orders   *ee.EntityController[Order]
	receipts *ee.EntityController[Receipt]
	items    *ee.EntityController[Item]
	payments *ee.EntityController[Payment]
	aggs     *ee.AggregateController[OrderAgg]
}

func newFixture(t *testing.T, opts ...ee.Option) *fixture {
	t.Helper()

	f := &fixture{store: mem.NewStore(), bus: mem.NewBus(mem.RetryBackoff(time.Millisecond), mem.MaxAttempts(3))}
	f.env = ee.MustNew(append([]ee.Option{ee.WithStore(f.store), ee.WithBus(f.bus), ee.WithOutboxPoll(10 * time.Millisecond)}, opts...)...)
	f.orders = must(ee.NewEntityController[Order](f.env))
	f.receipts = must(ee.NewEntityController[Receipt](f.env))
	f.items = must(ee.NewEntityController[Item](f.env))
	f.payments = must(ee.NewEntityController[Payment](f.env))
	f.aggs = must(ee.NewAggregateController[OrderAgg](f.env))

	t.Cleanup(func() {
		_ = f.env.Close()
		_ = f.bus.Close()
	})

	return f
}

func (f *fixture) start(t *testing.T) {
	t.Helper()

	if err := f.env.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}

	return v
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()

	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		if cond() {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}

func TestActionSaveEmitAndFilter(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	err := f.orders.Action(func(ctx *ee.ActionCtx[Order, OrderPaid]) error {
		o, err := ctx.Entity()
		if err != nil {
			return err
		}

		o.Status = "paid"
		if err := ctx.Save(o); err != nil {
			return err
		}

		return ctx.Emit(ReceiptRequested{OrderID: o.ID})
	}, ee.Filter(ee.Match("merchant", "m1")))
	if err != nil {
		t.Fatal(err)
	}

	err = f.receipts.Action(func(ctx *ee.ActionCtx[Receipt, ReceiptRequested]) error {
		return ctx.Create(&Receipt{ID: "r-" + ctx.Event().OrderID, OrderID: ctx.Event().OrderID})
	})
	if err != nil {
		t.Fatal(err)
	}

	f.start(t)

	for _, o := range []Order{{ID: "o1", Merchant: "m1"}, {ID: "o2", Merchant: "m2"}} {
		if err := f.orders.Create(ctx, o); err != nil {
			t.Fatal(err)
		}
	}

	_ = f.env.Send(ctx, OrderPaid{OrderID: "o1", Merchant: "m1"})
	_ = f.env.Send(ctx, OrderPaid{OrderID: "o2", Merchant: "m2"}) // filtered out

	eventually(t, "receipt r-o1", func() bool {
		_, err := f.receipts.Get(ctx, "r-o1")

		return err == nil
	})

	if o, _ := f.orders.Get(ctx, "o1"); o.Status != "paid" {
		t.Errorf("o1 status = %q", o.Status)
	}

	time.Sleep(50 * time.Millisecond)

	if o, _ := f.orders.Get(ctx, "o2"); o.Status != "" {
		t.Errorf("o2 must be filtered out, status = %q", o.Status)
	}

	if _, err := f.receipts.Get(ctx, "r-o2"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("r-o2: want ErrNotFound, got %v", err)
	}
}

func touchAction(done chan<- struct{}) func(ctx *ee.ActionCtx[Order, OrderTouched]) error {
	return func(ctx *ee.ActionCtx[Order, OrderTouched]) error {
		defer func() { done <- struct{}{} }()

		o, err := ctx.Entity()
		if err != nil {
			return err
		}

		o.Status = "touched" // no Save

		return nil
	}
}

func TestExplicitSaveVsAutoSave(t *testing.T) {
	for _, auto := range []bool{false, true} {
		t.Run(fmt.Sprint("auto=", auto), func(t *testing.T) {
			var opts []ee.Option
			if auto {
				opts = append(opts, ee.WithAutoSave())
			}

			f := newFixture(t, opts...)
			ctx := t.Context()
			done := make(chan struct{}, 1)

			_ = f.orders.Action(touchAction(done))
			f.start(t)

			_ = f.orders.Create(ctx, Order{ID: "o1"})
			_ = f.env.Send(ctx, OrderTouched{OrderID: "o1"})
			<-done

			want := ""
			if auto {
				want = "touched"
			}

			eventually(t, "status "+want, func() bool {
				o, _ := f.orders.Get(ctx, "o1")

				return o.Status == want
			})
		})
	}
}

func TestExpiredEventIsDropped(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	done := make(chan struct{}, 2)

	_ = f.orders.Action(touchAction(done))
	f.start(t)

	_ = f.orders.Create(ctx, Order{ID: "o1"})
	_ = f.env.Send(ctx, OrderTouched{OrderID: "o1"}, ee.ExpireAt(time.Now().Add(-time.Second)))
	_ = f.env.Send(ctx, OrderTouched{OrderID: "o1"}, ee.ExpireIn(time.Minute))

	<-done

	select {
	case <-done:
		t.Fatal("expired event was handled")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestFailedActionRollsBack(t *testing.T) {
	var (
		mu   sync.Mutex
		dead []string
	)

	f := newFixture(t)
	f.bus = mem.NewBus(mem.MaxAttempts(2), mem.RetryBackoff(time.Millisecond), mem.OnDeadLetter(func(e bus.Envelope, _ error) {
		mu.Lock()
		dead = append(dead, e.Event)
		mu.Unlock()
	}))
	f.env = ee.MustNew(ee.WithStore(f.store), ee.WithBus(f.bus))
	orders := must(ee.NewEntityController[Order](f.env))
	receipts := must(ee.NewEntityController[Receipt](f.env))
	t.Cleanup(func() { _ = f.env.Close(); _ = f.bus.Close() })

	ctx := t.Context()

	_ = orders.Action(func(ctx *ee.ActionCtx[Order, OrderPaid]) error {
		o, _ := ctx.Entity()
		o.Status = "paid"
		_ = ctx.Save(o)
		_ = ctx.Emit(ReceiptRequested{OrderID: o.ID})

		return errors.New("boom")
	})
	_ = receipts.Action(func(ctx *ee.ActionCtx[Receipt, ReceiptRequested]) error {
		t.Error("event emitted by a failed action was published")

		return nil
	})

	if err := f.env.Start(ctx); err != nil {
		t.Fatal(err)
	}

	_ = orders.Create(ctx, Order{ID: "o1"})
	_ = f.env.Send(ctx, OrderPaid{OrderID: "o1"})

	eventually(t, "dead letter", func() bool {
		mu.Lock()
		defer mu.Unlock()

		return slices.Equal(dead, []string{"order.paid"})
	})

	if o, _ := orders.Get(ctx, "o1"); o.Status != "" {
		t.Errorf("save of a failed action was committed: %q", o.Status)
	}
}

func TestConcurrentUpdatesRetryOnConflict(t *testing.T) {
	f := newFixture(t, ee.WithMaxRetries(100))
	ctx := context.Background()

	_ = f.orders.Create(ctx, Order{ID: "o1"})

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			err := f.orders.Update(ctx, "o1", func(o *Order) error {
				o.Amount++
				time.Sleep(time.Millisecond) // widen the race window

				return nil
			})
			if err != nil {
				t.Error(err)
			}
		})
	}

	wg.Wait()

	if o, _ := f.orders.Get(ctx, "o1"); o.Amount != 20 {
		t.Errorf("amount = %d, want 20 (lost updates)", o.Amount)
	}
}

func TestFind(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	for i := range 5 {
		_ = f.orders.Create(ctx, Order{ID: fmt.Sprint("o", i), Merchant: []string{"a", "b"}[i%2], Amount: i * 10})
	}

	var got []string

	for o, err := range f.orders.Find(ctx, ee.Where("Merchant", "a").And("Amount", ee.Gt, 0).OrderBy("Amount", true)) {
		if err != nil {
			t.Fatal(err)
		}

		got = append(got, o.ID)
	}

	if !slices.Equal(got, []string{"o4", "o2"}) {
		t.Errorf("got %v", got)
	}

	n := 0
	for range f.orders.Find(ctx, ee.Query{}.Limit(3)) {
		n++
	}

	if n != 3 {
		t.Errorf("limit: got %d", n)
	}

	for _, err := range f.orders.Find(ctx, ee.Where("Nope", 1)) {
		if !errors.Is(err, ee.ErrInvalidQuery) {
			t.Errorf("want ErrInvalidQuery, got %v", err)
		}
	}
}

func TestAggregateLifecycle(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()

	err := f.aggs.Create(ctx, &OrderAgg{
		Order:    &Order{ID: "o1"},
		Items:    []Item{{ID: "i1", OrderID: "o1"}, {ID: "i2", OrderID: "o1"}},
		Payments: map[string]Payment{"card": {ID: "p1", OrderID: "o1", Method: "card", Amount: 5}},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = f.aggs.Update(ctx, "o1", func(a *OrderAgg) error {
		a.Items = slices.DeleteFunc(a.Items, func(it Item) bool { return it.ID == "i1" })
		a.Items = append(a.Items, Item{ID: "i3", OrderID: "o1"})
		p := a.Payments["card"]
		p.Amount = 7
		a.Payments["card"] = p

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	a, err := f.aggs.Get(ctx, "o1")
	if err != nil {
		t.Fatal(err)
	}

	ids := []string{}
	for _, it := range a.Items {
		ids = append(ids, it.ID)
	}

	slices.Sort(ids)

	if !slices.Equal(ids, []string{"i2", "i3"}) || a.Payments["card"].Amount != 7 {
		t.Errorf("after update: items %v, payment %+v", ids, a.Payments["card"])
	}

	if _, err := f.items.Get(ctx, "i1"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("removed item must be deleted, got %v", err)
	}

	if err := f.aggs.Delete(ctx, "o1"); err != nil {
		t.Fatal(err)
	}

	for _, kind := range []string{"order", "item", "payment"} {
		if n := f.store.Len(kind); n != 0 {
			t.Errorf("%s: %d records left after Delete", kind, n)
		}
	}
}

func TestAggregateAction(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	done := make(chan struct{}, 1)

	err := f.aggs.Action(func(ctx *ee.AggregateCtx[OrderAgg, ItemAdded]) error {
		defer func() { done <- struct{}{} }()

		a, err := ctx.Aggregate()
		if err != nil {
			return err
		}

		e := ctx.Event()
		a.Items = append(a.Items, Item{ID: e.ItemID, OrderID: e.OrderID, SKU: e.SKU})
		a.Order.Amount++

		return ctx.Save(a)
	})
	if err != nil {
		t.Fatal(err)
	}

	f.start(t)

	_ = f.aggs.Create(ctx, &OrderAgg{Order: &Order{ID: "o1"}})
	_ = f.env.Send(ctx, ItemAdded{OrderID: "o1", ItemID: "i1", SKU: "x"})
	<-done

	eventually(t, "item i1", func() bool {
		it, err := f.items.Get(ctx, "i1")

		return err == nil && it.SKU == "x"
	})

	if o, _ := f.orders.Get(ctx, "o1"); o.Amount != 1 {
		t.Errorf("root not saved: %+v", o)
	}
}

func TestSaveRequiresLoadedEntity(t *testing.T) {
	f := newFixture(t)
	done := make(chan error, 1)

	_ = f.orders.Action(func(ctx *ee.ActionCtx[Order, OrderTouched]) error {
		done <- ctx.Save(&Order{ID: ctx.Event().OrderID})

		return nil
	})
	f.start(t)

	_ = f.orders.Create(t.Context(), Order{ID: "o1"})
	_ = f.env.Send(t.Context(), OrderTouched{OrderID: "o1"})

	if err := <-done; !errors.Is(err, ee.ErrNotLoaded) {
		t.Errorf("want ErrNotLoaded, got %v", err)
	}
}

func TestMiddlewareOrderAndSkip(t *testing.T) {
	var (
		mu    sync.Mutex
		trail []string
	)

	note := func(s string) {
		mu.Lock()
		trail = append(trail, s)
		mu.Unlock()
	}

	wrap := func(name string) ee.Middleware {
		return func(next ee.Next) ee.Next {
			return func(ctx context.Context, d *ee.Delivery) error {
				note(name + ">")
				defer note("<" + name)

				if d.Owner != "entity:order" || d.Event != "order.touched" || d.Tx == nil {
					t.Errorf("delivery = %+v", d)
				}

				if d.Headers["skip"] == "yes" {
					return nil
				}

				return next(ctx, d)
			}
		}
	}

	f := newFixture(t, ee.WithMiddleware(wrap("a"), wrap("b")))
	done := make(chan struct{}, 2)

	_ = f.orders.Action(func(ctx *ee.ActionCtx[Order, OrderTouched]) error {
		note("handler")
		done <- struct{}{}

		return nil
	})
	f.start(t)

	_ = f.env.Send(t.Context(), OrderTouched{OrderID: "o1"}, ee.WithHeader("skip", "yes"))
	_ = f.env.Send(t.Context(), OrderTouched{OrderID: "o1"})
	<-done

	mu.Lock()
	defer mu.Unlock()

	// "a" skips the first event, so "b" and the handler never see it.
	want := []string{"a>", "<a", "a>", "b>", "handler", "<b", "<a"}
	if !slices.Equal(trail, want) {
		t.Errorf("trail = %v", trail)
	}
}

func TestDedup(t *testing.T) {
	for _, dedup := range []bool{false, true} {
		t.Run(fmt.Sprint("dedup=", dedup), func(t *testing.T) {
			var opts []ee.Option
			if dedup {
				opts = append(opts, ee.WithMiddleware(ee.Dedup))
			}

			f := newFixture(t, opts...)
			ctx := t.Context()
			handled := make(chan struct{}, 2)

			_ = f.orders.Action(func(ctx *ee.ActionCtx[Order, OrderTouched]) error {
				o, err := ctx.Entity()
				if err != nil {
					return err
				}

				o.Amount++
				handled <- struct{}{}

				return ctx.Save(o)
			})
			f.start(t)

			_ = f.orders.Create(ctx, Order{ID: "o1"})

			// Simulate a redelivery: the same envelope arrives twice.
			payload, _ := json.Marshal(OrderTouched{OrderID: "o1"})
			env := bus.Envelope{ID: "dup-1", Event: "order.touched", PartitionKey: "o1", Payload: payload}
			_ = f.bus.Publish(ctx, env, env)

			want := 2
			if dedup {
				want = 1
			}

			eventually(t, fmt.Sprint("amount ", want), func() bool {
				o, _ := f.orders.Get(ctx, "o1")

				return o.Amount == want
			})

			time.Sleep(30 * time.Millisecond)

			if o, _ := f.orders.Get(ctx, "o1"); o.Amount != want {
				t.Errorf("amount = %d, want %d", o.Amount, want)
			}
		})
	}
}

type reserved struct {
	ID string `ee:"id"`
}

func (reserved) EntityName() string { return "ee.mine" }

func TestReservedEntityPrefix(t *testing.T) {
	if _, err := ee.NewEntityController[reserved](ee.MustNew()); !errors.Is(err, ee.ErrInvalidSchema) {
		t.Errorf("want ErrInvalidSchema, got %v", err)
	}
}
