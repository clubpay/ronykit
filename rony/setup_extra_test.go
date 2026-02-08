package rony

import (
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
)

type badAction struct{}

type badState struct{}

func (badState) Name() string {
	return "bad"
}

func (badState) Reduce(_ badAction) error {
	return nil
}

type goodAction struct{}

type goodState struct{}

func (*goodState) Name() string {
	return "good"
}

func (*goodState) Reduce(_ goodAction) error {
	return nil
}

type goodIn struct {
	ID int `json:"id"`
}

type goodOut struct {
	OK bool `json:"ok"`
}

func TestSetupPanicsOnNonPointerState(t *testing.T) {
	srv := NewServer()

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for non-pointer state")
		}
	}()

	Setup[badState, badAction](
		srv,
		"svc",
		func() badState { return badState{} },
	)
}

func TestSetupRegistersUnaryAndMiddleware(t *testing.T) {
	srv := NewServer()

	handler := func(_ *UnaryCtx[*goodState, goodAction], _ goodIn) (*goodOut, error) {
		return &goodOut{OK: true}, nil
	}

	stateless := func(_ *kit.Context) {}
	stateful := func(_ *BaseCtx[*goodState, goodAction]) {}
	selector := func(_ *kit.LimitedContext) (string, error) {
		return "node", nil
	}

	Setup[*goodState, goodAction](
		srv,
		"svc",
		func() *goodState { return &goodState{} },
		WithMiddleware[*goodState, goodAction](stateless),
		WithMiddleware[*goodState, goodAction](stateful),
		WithCoordinator[*goodState, goodAction](selector),
		WithUnary[*goodState, goodAction, goodIn, goodOut](
			handler,
			GET("/v1", UnaryName("Custom")),
			UnaryHeader(RequiredHeader("x-token")),
			UnaryMiddleware(func(_ *kit.Context) {}),
		),
	)

	svc := srv.cfg.services["svc"]
	if svc == nil {
		t.Fatal("expected service to be registered")
	}
	if len(svc.Contracts) != 1 {
		t.Fatalf("unexpected contracts count: %d", len(svc.Contracts))
	}

	contract := svc.Contracts[0]
	if contract.EdgeSelector == nil {
		t.Fatal("expected coordinator to be set")
	}
	if len(contract.RouteSelectors) != 1 {
		t.Fatalf("unexpected route selectors count: %d", len(contract.RouteSelectors))
	}
	if contract.RouteSelectors[0].Name != "Custom" {
		t.Fatalf("unexpected route name: %s", contract.RouteSelectors[0].Name)
	}
	if len(contract.InputHeaders) != 1 || contract.InputHeaders[0].Name != "x-token" {
		t.Fatalf("unexpected input headers: %#v", contract.InputHeaders)
	}
	if len(contract.Handlers) != 4 {
		t.Fatalf("unexpected handlers count: %d", len(contract.Handlers))
	}

	descs := srv.ExportDesc()
	if len(descs) != 1 {
		t.Fatalf("unexpected exported descs count: %d", len(descs))
	}
	if got := descs[0].Desc().Name; got != "svc" {
		t.Fatalf("unexpected exported service name: %s", got)
	}
}

func TestSetupWithContractAppliesMiddleware(t *testing.T) {
	srv := NewServer()

	contract := desc.NewContract().
		SetName("c1").
		SetHandler(func(_ *kit.Context) {})

	Setup[EMPTY, NOP](
		srv,
		"svc",
		EmptyState(),
		WithMiddleware[EMPTY, NOP](func(_ *kit.Context) {}),
		WithContract[EMPTY, NOP](contract),
	)

	svc := srv.cfg.services["svc"]
	if svc == nil || len(svc.Contracts) != 1 {
		t.Fatalf("unexpected contracts: %#v", svc)
	}

	if len(svc.Contracts[0].Handlers) != 2 {
		t.Fatalf("unexpected handlers count: %d", len(svc.Contracts[0].Handlers))
	}
}
