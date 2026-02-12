package fastws

import (
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	kiterrors "github.com/clubpay/ronykit/kit/errors"
)

type simpleMsg struct {
	Value string `json:"value"`
}

func TestSelectorHelpers(t *testing.T) {
	sel := RPC("ping").SetEncoding(kit.JSON)
	if sel.GetPredicate() != "ping" {
		t.Fatalf("unexpected predicate: %s", sel.GetPredicate())
	}
	if sel.GetEncoding() != kit.JSON {
		t.Fatalf("unexpected encoding: %v", sel.GetEncoding())
	}
	if sel.Query(queryPredicate) != "ping" {
		t.Fatalf("unexpected query result: %v", sel.Query(queryPredicate))
	}
	if sel.String() != "ping" {
		t.Fatalf("unexpected string: %s", sel.String())
	}

	selectors := RPCs("a", "b")
	if len(selectors) != 2 {
		t.Fatalf("unexpected selectors count: %d", len(selectors))
	}
}

func TestBundleDispatchErrors(t *testing.T) {
	gw, err := New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := gw.(*bundle)

	_, err = b.Dispatch(&kit.Context{}, nil)
	if err != kit.ErrDecodeIncomingContainerFailed {
		t.Fatalf("expected ErrDecodeIncomingContainerFailed, got: %v", err)
	}

	_, err = b.Dispatch(&kit.Context{}, []byte("{"))
	if !kiterrors.Is(err, kit.ErrDecodeIncomingMessageFailed) {
		t.Fatalf("expected ErrDecodeIncomingMessageFailed, got: %v", err)
	}

	out := common.SimpleOutgoingJSONRPC()
	out.SetID("1")
	out.SetHdr(b.predicateKey, "missing")
	out.InjectMessage(simpleMsg{Value: "v"})
	data, marshalErr := out.Marshal()
	if marshalErr != nil {
		t.Fatalf("marshal failed: %v", marshalErr)
	}
	out.Release()

	_, err = b.Dispatch(&kit.Context{}, data)
	if err != kit.ErrNoHandler {
		t.Fatalf("expected ErrNoHandler, got: %v", err)
	}
}

func TestBundleRegisterStoresRoute(t *testing.T) {
	gw, err := New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := gw.(*bundle)

	b.Register("svc", "contract", kit.JSON, RPC("evt"), simpleMsg{}, simpleMsg{})
	if _, ok := b.routes["evt"]; !ok {
		t.Fatal("expected route to be registered")
	}
}
