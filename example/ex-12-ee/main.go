// Example ex-12-ee: an order aggregate driven by events, on the in-memory
// bus and store. Swap mem for std/ee/store/postgres and std/ee/bus/nats (or
// kafka) in production; nothing else changes.
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/mem"
	"github.com/clubpay/ronykit/x/rkit"
)

// Order and OrderItem are states: plain structs with an `ee:"id"` field.
type Order struct {
	ID     string `ee:"id"`
	Store  string
	Status string
}

func (Order) StateName() string { return "order" }

type OrderItem struct {
	ID      string `ee:"id"`
	OrderID string
	Name    string
	Price   float64
}

func (OrderItem) StateName() string { return "order-item" }

// OrderAgg groups an order with its items. Every field is a member state;
// the root is written on every save, so concurrent saves of one order
// serialize.
type OrderAgg struct {
	Order *Order `ee:"root"`
	Items []OrderItem
}

func (OrderAgg) AggregateName() string { return "order-agg" }

func (a *OrderAgg) Load(ctx *ee.LoadCtx, id string) error {
	o, err := ctx.States[Order]().Get(id)
	if err != nil {
		return err
	}

	a.Order = o

	// Queries use Go field names.
	for it, err := range ctx.States[OrderItem]().Find(ee.Where("OrderID", id)) {
		if err != nil {
			return err
		}

		a.Items = append(a.Items, rkit.PtrVal(it))
	}

	return nil
}

func (a *OrderAgg) Total() float64 {
	var t float64
	for _, it := range a.Items {
		t += it.Price
	}

	return t
}

// Events name themselves; Key is the partition key and the id actions load.
type ItemAdded struct {
	OrderID, ItemID, Name string
	Price                 float64
}

func (ItemAdded) EventName() string { return "order.item-added" }
func (e ItemAdded) Key() string     { return e.OrderID }

type OrderSubmitted struct{ OrderID string }

func (OrderSubmitted) EventName() string { return "order.submitted" }
func (e OrderSubmitted) Key() string     { return e.OrderID }

func main() {
	ctx := context.Background()

	env := rkit.Must(ee.New(
		ee.WithBus(mem.NewBus()),
		ee.WithStore(mem.NewStore()),
	))

	// Register states before aggregates that use them, and everything before
	// Start.
	orders := rkit.Must(ee.NewStateController[Order](env))
	_ = rkit.Must(ee.NewStateController[OrderItem](env))
	orderAggs := rkit.Must(ee.NewAggregateController[OrderAgg](env))

	// Aggregate action: load the order with its items, add one, save the diff.
	must(orderAggs.Action(func(ctx *ee.AggregateCtx[OrderAgg, ItemAdded]) error {
		a, err := ctx.Aggregate()
		if err != nil {
			return err
		}

		e := ctx.Event()
		a.Items = append(a.Items, OrderItem{ID: e.ItemID, OrderID: e.OrderID, Name: e.Name, Price: e.Price})

		return ctx.Save(a)
	}))

	// State action: saves are explicit unless the Env uses ee.WithAutoSave.
	must(orders.Action(func(ctx *ee.ActionCtx[Order, OrderSubmitted]) error {
		o, err := ctx.State()
		if err != nil {
			return err
		}

		o.Status = "submitted"

		return ctx.Save(o)
	}))

	must(env.Start(ctx))

	defer func() { must(env.Close()) }()

	must(orderAggs.Create(ctx, &OrderAgg{Order: &Order{ID: "o1", Store: "s1", Status: "open"}}))
	must(env.Send(ctx, ItemAdded{OrderID: "o1", ItemID: "i1", Name: "coffee", Price: 3.5}))
	must(env.Send(ctx, ItemAdded{OrderID: "o1", ItemID: "i2", Name: "cake", Price: 4}))
	must(env.Send(ctx, OrderSubmitted{OrderID: "o1"}))

	// Actions run asynchronously; wait until both have landed.
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(10 * time.Millisecond) {
		a, err := orderAggs.Get(ctx, "o1")
		if err == nil && a.Order.Status == "submitted" && len(a.Items) == 2 {
			fmt.Printf("order %s: %s, %d items, total %.2f\n", a.Order.ID, a.Order.Status, len(a.Items), a.Total())

			return
		}

		if time.Now().After(deadline) {
			panic(fmt.Sprintf("timed out: %+v, %v", a, err))
		}
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
