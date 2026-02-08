package rony

import (
	"testing"

	"github.com/clubpay/ronykit/kit/desc"
)

type streamIn struct {
	ID int `json:"id"`
}

type streamOut struct {
	OK bool `json:"ok"`
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
