package rony

import (
	"errors"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
)

type rawIn struct {
	Payload string `json:"payload"`
}

func TestUnaryOptionHelpers(t *testing.T) {
	cfg := unaryConfig{}

	ALL("/all")(&cfg)
	POST("/post")(&cfg)
	PUT("/put")(&cfg)
	DELETE("/delete")(&cfg)
	PATCH("/patch")(&cfg)
	HEAD("/head")(&cfg)
	OPTIONS("/options")(&cfg)
	REST("GET", "/get",
		UnaryName("Custom"),
		UnaryDecoder(nil),
		UnaryDeprecated(true),
	)(&cfg)

	UnaryInputMeta(desc.WithField("f", desc.FieldMeta{Optional: true}))(&cfg)
	UnaryOutputMeta(desc.WithField("g", desc.FieldMeta{Deprecated: true}))(&cfg)
	UnaryHeader(RequiredHeader("x"), OptionalHeader("y"))(&cfg)
	UnaryMiddleware(func(*kit.Context) {})(&cfg)
	UnaryMiddlewareFn(func() StatelessMiddleware { return func(*kit.Context) {} })(&cfg)

	if len(cfg.Selectors) == 0 {
		t.Fatal("expected selectors to be set")
	}
	if len(cfg.Middlewares) == 0 {
		t.Fatal("expected middlewares to be set")
	}
	if len(cfg.Headers) != 2 {
		t.Fatalf("unexpected headers count: %d", len(cfg.Headers))
	}
	if cfg.InputMetaOptions == nil || cfg.OutputMetaOptions == nil {
		t.Fatal("expected meta options to be set")
	}
}

func TestSetupRawUnary(t *testing.T) {
	srv := NewServer()

	handler := func(_ *UnaryCtx[EMPTY, NOP], in kit.RawMessage) (kit.RawMessage, error) {
		return kit.RawMessage("ok"), nil
	}

	Setup[EMPTY, NOP](
		srv,
		"svc",
		EmptyState(),
		WithRawUnary[EMPTY, NOP, kit.RawMessage](
			handler,
			GET("/raw"),
		),
	)

	svc := srv.cfg.services["svc"]
	if svc == nil || len(svc.Contracts) != 1 {
		t.Fatalf("unexpected contracts: %#v", svc)
	}

	contract := svc.Contracts[0]
	err := kit.NewTestContext().
		Input(kit.RawMessage("in"), kit.EnvelopeHdr{}).
		SetHandler(contract.Handlers...).
		Expect(func(e *kit.Envelope) error {
			if string(e.GetMsg().(kit.RawMessage)) != "ok" { //nolint:forcetypeassert
				return errors.New("unexpected message")
			}

			return nil
		}).
		RunREST()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetupRawUnaryErrorPath(t *testing.T) {
	srv := NewServer()

	handler := func(_ *UnaryCtx[EMPTY, NOP], _ rawIn) (kit.RawMessage, error) {
		return nil, errors.New("boom")
	}

	Setup[EMPTY, NOP](
		srv,
		"svc",
		EmptyState(),
		WithRawUnary[EMPTY, NOP, rawIn](
			handler,
			GET("/raw"),
		),
	)

	svc := srv.cfg.services["svc"]
	contract := svc.Contracts[0]
	err := kit.NewTestContext().
		Input(&rawIn{Payload: "x"}, kit.EnvelopeHdr{}).
		SetHandler(contract.Handlers...).
		Receiver(func(out ...*kit.Envelope) error {
			if len(out) == 0 {
				return errors.New("expected error response")
			}

			return nil
		}).
		RunREST()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
