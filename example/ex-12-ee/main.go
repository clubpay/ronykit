// Example ex-12-ee: an order/basket service. rony serves the HTTP API and ee
// keeps the state: items, per-item inventory, orders and order lines.
//
// Adding an item to an order reserves stock; checkout submits the order and
// deducts the reserved stock, atomically. The example runs on ee's in-memory
// bus and store; swap mem for std/ee/store/postgres and std/ee/bus/nats (or
// kafka) in production — nothing else changes.
package main

import (
	"context"
	"os"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/mem"
	"github.com/clubpay/ronykit/rony"
	"github.com/clubpay/ronykit/x/rkit"
)

func main() {
	ctx := context.Background()

	env := rkit.Must(ee.New(
		ee.WithBus(mem.NewBus()),
		ee.WithStore(mem.NewStore()),
	))

	// Register everything before Start.
	shop := rkit.Must(NewShop(env))

	rkit.Assert(env.Start(ctx))

	defer func() { rkit.Assert(env.Close()) }()

	rkit.Assert(seed(ctx, shop))

	srv := rony.NewServer(
		rony.Listen(":8080"),
		rony.WithServerName("ShopServer"),
		rony.WithAPIDocs("/docs"),
		rony.UseScalarUI(),
	)
	shop.Setup(srv)

	rkit.Assert(srv.Run(ctx, os.Kill, os.Interrupt))
}

func seed(ctx context.Context, s *Shop) error {
	catalog := []struct {
		Item
		onHand int
	}{
		{Item{ID: "coffee", Name: "Coffee", Price: 350}, 20},
		{Item{ID: "cake", Name: "Cake", Price: 400}, 5},
		{Item{ID: "sandwich", Name: "Sandwich", Price: 650}, 0},
	}

	for _, c := range catalog {
		if err := s.items.Create(ctx, c.Item); err != nil {
			return err
		}

		if err := s.inventory.Create(ctx, Inventory{ID: c.ID, OnHand: c.onHand}); err != nil {
			return err
		}
	}

	return nil
}

