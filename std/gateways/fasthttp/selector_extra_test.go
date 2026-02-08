package fasthttp

import (
	"testing"

	"github.com/clubpay/ronykit/kit"
)

func TestSelectorQueryAndString(t *testing.T) {
	dec := func(_ *RequestCtx, _ []byte) (kit.Message, error) { return kit.RawMessage("ok"), nil }
	sel := REST(MethodGet, "/path").SetDecoder(dec).SetEncoding(kit.JSON)

	if sel.Query(queryMethod) != MethodGet {
		t.Fatalf("expected method query to return %s", MethodGet)
	}
	if sel.Query(queryPath) != "/path" {
		t.Fatalf("expected path query to return /path")
	}
	if sel.Query(queryDecoder) == nil {
		t.Fatalf("expected decoder query to return value")
	}
	if sel.Query("unknown") != nil {
		t.Fatalf("expected unknown query to return nil")
	}
	if sel.String() != "GET /path" {
		t.Fatalf("unexpected selector string: %s", sel.String())
	}

	rpc := RPC("pred")
	if rpc.String() != "pred" {
		t.Fatalf("unexpected rpc selector string: %s", rpc.String())
	}

	if sel.GetEncoding() != kit.JSON {
		t.Fatalf("expected encoding to be set")
	}
}

func TestRPCs(t *testing.T) {
	sels := RPCs("a", "b", "c")
	if len(sels) != 3 {
		t.Fatalf("expected 3 selectors, got %d", len(sels))
	}
	if sels[0].(Selector).Predicate != "a" { //nolint:forcetypeassert
		t.Fatalf("unexpected predicate: %v", sels[0])
	}
}
