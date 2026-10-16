package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/mem"
	"github.com/clubpay/ronykit/rony"
	"github.com/stretchr/testify/require"
)

// Seeded stock (see seed): coffee 20, cake 5, sandwich 0.

// An item already in the order is merged into its line, and the extra
// quantity is reserved on top of what the line already holds.
func TestAddItemToExistingLine(t *testing.T) {
	c := startShop(t)

	order := c.createOrder()
	c.addItem(order, "coffee", 2).expect(http.StatusOK)

	res := c.addItem(order, "coffee", 3).expect(http.StatusOK)

	var o OrderDTO
	res.decode(&o)
	require.Equal(t, OrderOpen, o.Status)
	require.Len(t, o.Lines, 1)
	require.Equal(t, "coffee", o.Lines[0].ItemID)
	require.Equal(t, 5, o.Lines[0].Quantity)
	require.Equal(t, int64(5*350), o.Total)

	require.Equal(t, 15, c.available("coffee"))
	require.Equal(t, InventoryDTO{ItemID: "coffee", OnHand: 20, Reserved: 5, Available: 15}, c.inventory("coffee"))
}

// Units reserved by an open order cannot be added to another order, even if
// the first order never checks out.
func TestReservedItemsBlockOtherOrders(t *testing.T) {
	c := startShop(t)

	first := c.createOrder()
	c.addItem(first, "cake", 4).expect(http.StatusOK)

	second := c.createOrder()
	c.addItem(second, "cake", 2).expect(http.StatusBadRequest).requireMsg("OUT_OF_STOCK")
	require.Empty(t, c.getOrder(second).Lines, "a rejected add must not leave a line")

	// What is left can still be taken.
	c.addItem(second, "cake", 1).expect(http.StatusOK)
	require.Equal(t, 0, c.available("cake"))
	require.Equal(t, InventoryDTO{ItemID: "cake", OnHand: 5, Reserved: 5, Available: 0}, c.inventory("cake"))

	// Releasing the first order's reservation frees the units again.
	c.do(http.MethodDelete, "/orders/"+first+"/items/cake", nil).expect(http.StatusOK)
	require.Equal(t, 4, c.available("cake"))
	c.addItem(second, "cake", 2).expect(http.StatusOK)
}

// Checkout turns the reservation into a stock deduction: the units are gone
// from OnHand, no longer reserved, and still unavailable to other orders.
func TestCheckoutDeductsInventory(t *testing.T) {
	c := startShop(t)

	first := c.createOrder()
	c.addItem(first, "cake", 4).expect(http.StatusOK)
	c.addItem(first, "coffee", 3).expect(http.StatusOK)

	second := c.createOrder()
	c.addItem(second, "cake", 2).expect(http.StatusBadRequest).requireMsg("OUT_OF_STOCK")

	var paid OrderDTO
	c.do(http.MethodPost, "/orders/"+first+"/checkout", nil).expect(http.StatusOK).decode(&paid)
	require.Equal(t, OrderSubmitted, paid.Status)
	require.NotNil(t, paid.SubmittedAt)
	require.Equal(t, int64(4*400+3*350), paid.Total)

	require.Equal(t, InventoryDTO{ItemID: "cake", OnHand: 1, Reserved: 0, Available: 1}, c.inventory("cake"))
	require.Equal(t, InventoryDTO{ItemID: "coffee", OnHand: 17, Reserved: 0, Available: 17}, c.inventory("coffee"))
	require.Equal(t, 1, c.available("cake"))

	// Sold units stay sold for the other order; the paid order is closed.
	c.addItem(second, "cake", 2).expect(http.StatusBadRequest).requireMsg("OUT_OF_STOCK")
	c.addItem(second, "cake", 1).expect(http.StatusOK)
	c.addItem(first, "coffee", 1).expect(http.StatusBadRequest).requireMsg("ORDER_NOT_OPEN")
	c.do(http.MethodPost, "/orders/"+first+"/checkout", nil).expect(http.StatusBadRequest).requireMsg("ORDER_NOT_OPEN")
}

type shopClient struct {
	t    *testing.T
	base string
	shop *Shop
}

// startShop runs the service on a fresh in-memory env and a free port.
func startShop(t *testing.T) *shopClient {
	t.Helper()

	ctx := context.Background()

	env, err := ee.New(ee.WithBus(mem.NewBus()), ee.WithStore(mem.NewStore()))
	require.NoError(t, err)

	shop, err := NewShop(env)
	require.NoError(t, err)
	require.NoError(t, env.Start(ctx))
	t.Cleanup(func() { require.NoError(t, env.Close()) })
	require.NoError(t, seed(ctx, shop))

	addr := freeAddr(t)
	srv := rony.NewServer(rony.Listen(addr))
	shop.Setup(srv)
	require.NoError(t, srv.Start(ctx))
	t.Cleanup(func() { srv.Stop(ctx) })

	c := &shopClient{t: t, base: "http://" + addr, shop: shop}
	require.Eventually(t, func() bool {
		res, err := http.Get(c.base + "/items")
		if err != nil {
			return false
		}

		_ = res.Body.Close()

		return res.StatusCode == http.StatusOK
	}, 5*time.Second, 20*time.Millisecond, "server did not come up")

	return c
}

func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := l.Addr().String()
	require.NoError(t, l.Close())

	return addr
}

type response struct {
	t      *testing.T
	status int
	body   []byte
}

func (c *shopClient) do(method, path string, body any) *response {
	c.t.Helper()

	var in io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(c.t, err)

		in = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.base+path, in)
	require.NoError(c.t, err)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	require.NoError(c.t, err)

	defer res.Body.Close()

	out, err := io.ReadAll(res.Body)
	require.NoError(c.t, err)

	return &response{t: c.t, status: res.StatusCode, body: out}
}

func (r *response) expect(status int) *response {
	r.t.Helper()
	require.Equal(r.t, status, r.status, "body: %s", r.body)

	return r
}

func (r *response) decode(v any) {
	r.t.Helper()
	require.NoError(r.t, json.Unmarshal(r.body, v), "body: %s", r.body)
}

func (r *response) requireMsg(msg string) {
	r.t.Helper()
	require.Contains(r.t, string(r.body), msg)
}

func (c *shopClient) createOrder() string {
	c.t.Helper()

	var o OrderDTO
	c.do(http.MethodPost, "/orders", struct{}{}).expect(http.StatusOK).decode(&o)
	require.NotEmpty(c.t, o.ID)

	return o.ID
}

func (c *shopClient) getOrder(id string) OrderDTO {
	c.t.Helper()

	var o OrderDTO
	c.do(http.MethodGet, "/orders/"+id, nil).expect(http.StatusOK).decode(&o)

	return o
}

func (c *shopClient) addItem(orderID, itemID string, qty int) *response {
	c.t.Helper()

	return c.do(
		http.MethodPost,
		"/orders/"+orderID+"/items",
		map[string]any{"itemId": itemID, "quantity": qty},
	)
}

// available reads an item's available stock from the public item list.
func (c *shopClient) available(itemID string) int {
	c.t.Helper()

	var res ListItemsResponse
	c.do(http.MethodGet, "/items", nil).expect(http.StatusOK).decode(&res)

	for _, it := range res.Items {
		if it.ID == itemID {
			return it.Available
		}
	}

	require.FailNow(c.t, fmt.Sprintf("item %q not listed", itemID))

	return 0
}

// inventory reads the stored inventory: there is no endpoint exposing
// OnHand and Reserved, which are what the scenarios are about.
func (c *shopClient) inventory(itemID string) InventoryDTO {
	c.t.Helper()

	inv, err := c.shop.inventory.Get(context.Background(), itemID)
	require.NoError(c.t, err)

	return InventoryDTO{ItemID: inv.ID, OnHand: inv.OnHand, Reserved: inv.Reserved, Available: inv.Available()}
}
