package rony

import (
	"errors"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
)

type streamIn struct {
	ID int `json:"id"`
}

type streamOut struct {
	OK bool `json:"ok"`
}

func TestSetupStreamSSEOption(t *testing.T) {
	srv := NewServer()

	handler := func(ctx *StreamCtx[EMPTY, NOP, streamOut], in streamIn) error {
		ctx.Push(streamOut{OK: true})

		return nil
	}

	opt := SetupOptionGroup[EMPTY, NOP](
		WithStream[EMPTY, NOP, streamIn, streamOut](
			handler,
			SSE("/stream"),
		),
	)

	Setup[EMPTY, NOP](srv, "svc", EmptyState(), opt)

	svc := srv.cfg.services["svc"]
	if svc == nil || len(svc.Contracts) != 1 {
		t.Fatalf("unexpected contracts: %#v", svc)
	}

	sel := svc.Contracts[0].RouteSelectors[0].Selector
	ss, ok := sel.(interface{ IsStream() bool })
	if !ok || !ss.IsStream() {
		t.Fatalf("expected SSE stream selector, got %#v", sel)
	}
}
func TestSetupStreamOptions(t *testing.T) {
	srv := NewServer()

	handler := func(ctx *StreamCtx[EMPTY, NOP, streamOut], in streamIn) error {
		ctx.Push(streamOut{OK: true})

		return nil
	}

	opt := SetupOptionGroup[EMPTY, NOP](
		WithStream[EMPTY, NOP, streamIn, streamOut](
			handler,
			RPC("pred"),
			StreamInputMeta(desc.WithField("id", desc.FieldMeta{Optional: true})),
			StreamOutputMeta(desc.WithField("ok", desc.FieldMeta{Deprecated: true})),
		),
	)

	Setup[EMPTY, NOP](srv, "svc", EmptyState(), opt)

	svc := srv.cfg.services["svc"]
	if svc == nil || len(svc.Contracts) != 1 {
		t.Fatalf("unexpected contracts: %#v", svc)
	}
	contract := svc.Contracts[0]
	if len(contract.RouteSelectors) != 1 {
		t.Fatalf("unexpected selectors count: %d", len(contract.RouteSelectors))
	}
	if contract.InputMeta.Fields["id"].Optional != true {
		t.Fatalf("unexpected input meta: %#v", contract.InputMeta.Fields)
	}
	if contract.OutputMeta.Fields["ok"].Deprecated != true {
		t.Fatalf("unexpected output meta: %#v", contract.OutputMeta.Fields)
	}
}

func TestSetupStreamErrorDoesNotSendEnvelope(t *testing.T) {
	srv := NewServer()

	handler := func(_ *StreamCtx[EMPTY, NOP, streamOut], _ streamIn) error {
		return errors.New("boom")
	}

	Setup[EMPTY, NOP](
		srv,
		"svc",
		EmptyState(),
		WithStream[EMPTY, NOP, streamIn, streamOut](
			handler,
			RPC("pred"),
		),
	)

	svc := srv.cfg.services["svc"]
	if svc == nil || len(svc.Contracts) != 1 {
		t.Fatalf("unexpected contracts: %#v", svc)
	}

	err := kit.NewTestContext().
		Input(&streamIn{ID: 1}, kit.EnvelopeHdr{}).
		SetHandler(svc.Contracts[0].Handlers...).
		Receiver(func(out ...*kit.Envelope) error {
			if len(out) != 0 {
				return errors.New("stream handler errors must not auto-send an envelope")
			}

			return nil
		}).
		Run(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
