package main

import (
	"errors"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/rony"
	"github.com/clubpay/ronykit/rony/errs"
	"github.com/clubpay/ronykit/x/rkit"
)

// Shop serves the HTTP API on top of the ee controllers. Every write is one
// ee unit of work, so a request either fully applies or not at all.
type Shop struct {
	env         *ee.Env
	items       *ee.StateController[Item]
	inventory   *ee.StateController[Inventory]
	orders      *ee.StateController[Order]
	basketLines *ee.AggregateController[BasketLine]
	orderAggs   *ee.AggregateController[OrderAgg]
}

func NewShop(env *ee.Env) (*Shop, error) {
	var (
		s   = &Shop{env: env}
		err error
	)

	// States first: aggregates check their members are registered.
	if s.items, err = ee.NewStateController[Item](env); err != nil {
		return nil, err
	}

	if s.inventory, err = ee.NewStateController[Inventory](env); err != nil {
		return nil, err
	}

	if s.orders, err = ee.NewStateController[Order](env); err != nil {
		return nil, err
	}

	if _, err = ee.NewStateController[OrderLine](env); err != nil {
		return nil, err
	}

	if s.basketLines, err = ee.NewAggregateController[BasketLine](env); err != nil {
		return nil, err
	}

	if s.orderAggs, err = ee.NewAggregateController[OrderAgg](env); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Shop) Setup(srv *rony.Server) {
	rony.Setup(
		srv,
		"ShopService",
		rony.EmptyState(),
		rony.WithUnary(s.listItems, rony.GET("/items")),
		rony.WithUnary(s.updateInventory, rony.PUT("/items/{id}/inventory")),
		rony.WithUnary(s.createOrder, rony.POST("/orders")),
		rony.WithUnary(s.getOrder, rony.GET("/orders/{id}")),
		rony.WithUnary(s.addItem, rony.POST("/orders/{id}/items")),
		rony.WithUnary(s.removeItem, rony.DELETE("/orders/{id}/items/{itemId}")),
		rony.WithUnary(s.checkout, rony.POST("/orders/{id}/checkout")),
	)
}

type (
	ItemDTO struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Price     int64  `json:"price"`
		Available int    `json:"available"`
	}
	ListItemsRequest  struct{}
	ListItemsResponse struct {
		Items []ItemDTO `json:"items"`
	}
	UpdateInventoryRequest struct {
		ID     string `json:"id"`
		OnHand int    `json:"onHand"`
	}
	InventoryDTO struct {
		ItemID    string `json:"itemId"`
		OnHand    int    `json:"onHand"`
		Reserved  int    `json:"reserved"`
		Available int    `json:"available"`
	}
	CreateOrderRequest struct{}
	OrderRequest       struct {
		ID string `json:"id"`
	}
	AddItemRequest struct {
		ID       string `json:"id"`
		ItemID   string `json:"itemId"`
		Quantity int    `json:"quantity"`
	}
	RemoveItemRequest struct {
		ID     string `json:"id"`
		ItemID string `json:"itemId"`
	}
	OrderLineDTO struct {
		ItemID   string `json:"itemId"`
		Name     string `json:"name"`
		Price    int64  `json:"price"`
		Quantity int    `json:"quantity"`
		Subtotal int64  `json:"subtotal"`
	}
	OrderDTO struct {
		ID          string         `json:"id"`
		Status      string         `json:"status"`
		CreatedAt   time.Time      `json:"createdAt"`
		SubmittedAt *time.Time     `json:"submittedAt,omitempty"`
		Lines       []OrderLineDTO `json:"lines"`
		Total       int64          `json:"total"`
	}
)

func (s *Shop) listItems(ctx *rony.SUnaryCtx, _ ListItemsRequest) (*ListItemsResponse, error) {
	stock := map[string]int{}

	for inv, err := range s.inventory.Find(ctx.Context(), ee.Query{}) {
		if err != nil {
			return nil, apiErr(err)
		}

		stock[inv.ID] = inv.Available()
	}

	res := &ListItemsResponse{Items: []ItemDTO{}}

	for it, err := range s.items.Find(ctx.Context(), ee.Query{}.OrderBy("Name", false)) {
		if err != nil {
			return nil, apiErr(err)
		}

		res.Items = append(
			res.Items,
			ItemDTO{ID: it.ID, Name: it.Name, Price: it.Price, Available: stock[it.ID]},
		)
	}

	return res, nil
}

// updateInventory is the back-office stock count: it sets the units on hand.
// Units reserved by open orders stay reserved, so the count cannot go below
// them.
func (s *Shop) updateInventory(ctx *rony.SUnaryCtx, req UpdateInventoryRequest) (*InventoryDTO, error) {
	if req.OnHand < 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("INVALID_ON_HAND").Err()
	}

	var out Inventory

	err := s.inventory.Update(ctx.Context(), req.ID, func(inv *Inventory) error {
		if req.OnHand < inv.Reserved {
			return errs.B().Code(errs.FailedPrecondition).
				Msgf("BELOW_RESERVED: %d units are reserved by open orders", inv.Reserved).Err()
		}

		inv.OnHand = req.OnHand
		out = *inv

		return nil
	})
	if err != nil {
		return nil, apiErr(err)
	}

	return &InventoryDTO{
		ItemID:    out.ID,
		OnHand:    out.OnHand,
		Reserved:  out.Reserved,
		Available: out.Available(),
	}, nil
}

func (s *Shop) createOrder(ctx *rony.SUnaryCtx, _ CreateOrderRequest) (*OrderDTO, error) {
	o := Order{ID: rkit.RandomID(12), Status: OrderOpen, CreatedAt: time.Now().UTC()}
	if err := s.orders.Create(ctx.Context(), o); err != nil {
		return nil, apiErr(err)
	}

	return toOrderDTO(&OrderAgg{Order: &o}), nil
}

func (s *Shop) getOrder(ctx *rony.SUnaryCtx, req OrderRequest) (*OrderDTO, error) {
	a, err := s.orderAggs.Get(ctx.Context(), req.ID)
	if err != nil {
		return nil, apiErr(err)
	}

	return toOrderDTO(a), nil
}

// addItem reserves Quantity more units of the item for the order, adding the
// line if the item is not in the order yet.
func (s *Shop) addItem(ctx *rony.SUnaryCtx, req AddItemRequest) (*OrderDTO, error) {
	if req.Quantity <= 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("INVALID_QUANTITY").Err()
	}

	item, err := s.items.Get(ctx.Context(), req.ItemID)
	if err != nil {
		return nil, apiErr(err)
	}

	err = s.basketLines.Update(ctx.Context(), lineID(req.ID, req.ItemID), func(b *BasketLine) error {
		if b.Order.Status != OrderOpen {
			return errOrderNotOpen
		}

		if b.Inventory.Available() < req.Quantity {
			return errs.B().Code(errs.FailedPrecondition).
				Msgf("OUT_OF_STOCK: only %d units of %s are available", b.Inventory.Available(), item.Name).
				Err()
		}

		b.Inventory.Reserved += req.Quantity

		if b.Line == nil {
			b.Line = &OrderLine{
				ID:      lineID(req.ID, req.ItemID),
				OrderID: req.ID,
				ItemID:  item.ID,
				Name:    item.Name,
				Price:   item.Price,
			}
		}

		b.Line.Quantity += req.Quantity

		return nil
	})
	if err != nil {
		return nil, apiErr(err)
	}

	return s.getOrder(ctx, OrderRequest{ID: req.ID})
}

// removeItem drops the line and releases its reservation.
func (s *Shop) removeItem(ctx *rony.SUnaryCtx, req RemoveItemRequest) (*OrderDTO, error) {
	err := s.basketLines.Update(ctx.Context(), lineID(req.ID, req.ItemID), func(b *BasketLine) error {
		if b.Order.Status != OrderOpen {
			return errOrderNotOpen
		}

		if b.Line == nil {
			return errs.B().Code(errs.NotFound).Msg("ITEM_NOT_IN_ORDER").Err()
		}

		b.Inventory.Reserved -= b.Line.Quantity
		b.Line = nil // the aggregate deletes members that disappear

		return nil
	})
	if err != nil {
		return nil, apiErr(err)
	}

	return s.getOrder(ctx, OrderRequest{ID: req.ID})
}

// checkout submits the order and turns every reservation into a stock
// deduction, all in one transaction.
func (s *Shop) checkout(ctx *rony.SUnaryCtx, req OrderRequest) (*OrderDTO, error) {
	var out *OrderAgg

	err := s.orderAggs.Update(ctx.Context(), req.ID, func(a *OrderAgg) error {
		if a.Order.Status != OrderOpen {
			return errOrderNotOpen
		}

		if len(a.Lines) == 0 {
			return errs.B().Code(errs.FailedPrecondition).Msg("EMPTY_ORDER").Err()
		}

		for _, l := range a.Lines {
			inv := a.Stock[l.ItemID]
			inv.Reserved -= l.Quantity
			inv.OnHand -= l.Quantity
		}

		a.Order.Status = OrderSubmitted
		a.Order.SubmittedAt = time.Now().UTC()
		out = a

		return nil
	})
	if err != nil {
		return nil, apiErr(err)
	}

	return toOrderDTO(out), nil
}

var errOrderNotOpen = errs.B().Code(errs.FailedPrecondition).Msg("ORDER_NOT_OPEN").Err()

// apiErr passes API errors through and maps ee errors to API codes.
func apiErr(err error) error {
	if _, ok := errors.AsType[*errs.Error](err); ok {
		return err
	}

	switch {
	case errors.Is(err, ee.ErrNotFound), errors.Is(err, errInvalidLineID):
		return errs.WrapCode(err, errs.NotFound, "NOT_FOUND")
	case errors.Is(err, ee.ErrConflict):
		return errs.WrapCode(err, errs.Aborted, "CONCURRENT_UPDATE")
	default:
		return errs.WrapCode(err, errs.Internal, "INTERNAL")
	}
}

func toOrderDTO(a *OrderAgg) *OrderDTO {
	o := &OrderDTO{
		ID:        a.Order.ID,
		Status:    a.Order.Status,
		CreatedAt: a.Order.CreatedAt,
		Lines:     make([]OrderLineDTO, 0, len(a.Lines)),
		Total:     a.Total(),
	}
	if !a.Order.SubmittedAt.IsZero() {
		o.SubmittedAt = &a.Order.SubmittedAt
	}

	for _, l := range a.Lines {
		o.Lines = append(o.Lines, OrderLineDTO{
			ItemID:   l.ItemID,
			Name:     l.Name,
			Price:    l.Price,
			Quantity: l.Quantity,
			Subtotal: l.Price * int64(l.Quantity),
		})
	}

	return o
}
