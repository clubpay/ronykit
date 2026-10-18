package main

import (
	"errors"
	"strings"
	"time"

	"github.com/clubpay/ronykit/ee"
)

// Item is a catalog entry. Prices are in minor units (cents).
type Item struct {
	ID    string `ee:"id"`
	Name  string
	Price int64
}

func (Item) EntityName() string { return "item" }

// Inventory is the stock of one item; its ID is the item ID. OnHand counts
// physical units, Reserved the units held by open orders.
type Inventory struct {
	ID       string `ee:"id"`
	OnHand   int
	Reserved int
}

func (Inventory) EntityName() string { return "inventory" }

func (i *Inventory) Available() int { return i.OnHand - i.Reserved }

const (
	OrderOpen      = "open"
	OrderSubmitted = "submitted"
)

type Order struct {
	ID          string `ee:"id"`
	Status      string
	CreatedAt   time.Time
	SubmittedAt time.Time
}

func (Order) EntityName() string { return "order" }

// OrderLine is one item in an order. While the order is open, its Quantity is
// reserved on the item's Inventory; checkout turns the reservation into a
// stock deduction. Name and Price are copied from the item when it is added.
type OrderLine struct {
	ID       string `ee:"id"`
	OrderID  string
	ItemID   string
	Name     string
	Price    int64
	Quantity int
}

func (OrderLine) EntityName() string { return "order-line" }

var errInvalidLineID = errors.New("invalid order line id")

func lineID(orderID, itemID string) string { return orderID + ":" + itemID }

func splitLineID(id string) (orderID, itemID string, ok bool) { return strings.Cut(id, ":") }

// BasketLine is the consistency boundary for changing one line of an order:
// the order, the item's inventory and the line are read and written in one
// transaction. The order is the root, so it is written on every save and
// serializes against checkout; Inventory carries its own version, so two
// orders reserving the same item serialize too. On conflict the whole update
// re-runs and re-checks the stock.
type BasketLine struct {
	Order     *Order `ee:"root"`
	Inventory *Inventory
	Line      *OrderLine // nil if the item is not in the order
}

func (BasketLine) AggregateName() string { return "basket-line" }

// Load takes "<orderID>:<itemID>".
func (b *BasketLine) Load(ctx *ee.LoadCtx, id string) error {
	orderID, itemID, ok := splitLineID(id)
	if !ok {
		return errInvalidLineID
	}

	var err error

	b.Order, err = ctx.Entities[Order]().Get(orderID)
	if err != nil {
		return err
	}

	b.Inventory, err = ctx.Entities[Inventory]().Get(itemID)
	if err != nil {
		return err
	}

	b.Line, err = ctx.Entities[OrderLine]().Get(id)
	if errors.Is(err, ee.ErrNotFound) {
		return nil
	}

	return err
}

// OrderAgg is a whole order: its lines and the inventory of every item in
// it. Checkout runs on it so that submitting the order and deducting all its
// stock happen atomically.
type OrderAgg struct {
	Order *Order `ee:"root"`
	Lines []OrderLine
	Stock map[string]*Inventory // by item ID
}

func (OrderAgg) AggregateName() string { return "order" }

func (a *OrderAgg) Load(ctx *ee.LoadCtx, id string) error {
	o, err := ctx.Entities[Order]().Get(id)
	if err != nil {
		return err
	}

	a.Order = o
	a.Stock = map[string]*Inventory{}

	for l, err := range ctx.Entities[OrderLine]().Find(ee.Where("OrderID", id).OrderBy("ItemID", false)) {
		if err != nil {
			return err
		}

		a.Lines = append(a.Lines, *l)
	}

	for _, l := range a.Lines {
		inv, err := ctx.Entities[Inventory]().Get(l.ItemID)
		if err != nil {
			return err
		}

		a.Stock[l.ItemID] = inv
	}

	return nil
}

func (a *OrderAgg) Total() int64 {
	var t int64
	for _, l := range a.Lines {
		t += l.Price * int64(l.Quantity)
	}

	return t
}
