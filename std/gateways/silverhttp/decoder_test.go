package silverhttp

import (
	"math"
	"strconv"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/silverhttp/httpmux"
)

type embeddedMsg struct {
	X int `json:"x"`
}

type sampleMsg struct {
	embeddedMsg
	Name    string  `json:"name"`
	Count   *int    `json:"count"`
	Enabled *bool   `json:"enabled"`
	Data    []byte  `json:"data"`
	Score   float64 `json:"score"`
}

func TestReflectDecoderPopulatesFields(t *testing.T) {
	dec := reflectDecoder(kit.JSON, kit.CreateMessageFactory(&sampleMsg{}))
	bag := httpmux.Params{
		{Key: "name", Value: "param"},
		{Key: "x", Value: "7"},
		{Key: "count", Value: "9"},
		{Key: "enabled", Value: "true"},
		{Key: "data", Value: "blob"},
		{Key: "score", Value: "3.5"},
	}

	msg, err := dec(nil, bag, []byte(`{"name":"body","score":1.2}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(*sampleMsg) //nolint:forcetypeassert

	if m.Name != "param" {
		t.Fatalf("unexpected name: %s", m.Name)
	}
	if m.X != 7 {
		t.Fatalf("unexpected embedded field: %d", m.X)
	}
	if m.Count == nil || *m.Count != 9 {
		t.Fatalf("unexpected count: %#v", m.Count)
	}
	if m.Enabled == nil || *m.Enabled != true {
		t.Fatalf("unexpected enabled: %#v", m.Enabled)
	}
	if string(m.Data) != "blob" {
		t.Fatalf("unexpected data: %s", string(m.Data))
	}
	if m.Score != 3.5 {
		t.Fatalf("unexpected score: %v", m.Score)
	}
}

func TestReflectDecoderPanicsOnInvalidFactory(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for non-pointer factory")
		}
	}()

	_ = reflectDecoder(kit.JSON, func() kit.Message { return sampleMsg{} })
}

func TestSelectorQueries(t *testing.T) {
	sel := REST(MethodGet, "/path").SetEncoding(kit.JSON)
	if sel.String() != MethodGet+" /path" {
		t.Fatalf("unexpected selector string: %s", sel.String())
	}
	if sel.Query(queryMethod) != MethodGet {
		t.Fatalf("unexpected query method: %v", sel.Query(queryMethod))
	}
	if sel.Query(queryPath) != "/path" {
		t.Fatalf("unexpected query path: %v", sel.Query(queryPath))
	}

	rpc := RPC("evt")
	if rpc.Query(queryPredicate) != "evt" {
		t.Fatalf("unexpected query predicate: %v", rpc.Query(queryPredicate))
	}
}

type wideIntMsg struct {
	Int    int     `json:"int"`
	IntPtr *int    `json:"intPtr"`
	Uint   uint    `json:"uint"`
	U32    uint32  `json:"u32"`
	U64    uint64  `json:"u64"`
	U64Ptr *uint64 `json:"u64Ptr"`
}

func TestReflectDecoderWideIntegers(t *testing.T) {
	if strconv.IntSize != 64 {
		t.Skip("requires 64-bit int")
	}

	dec := reflectDecoder(kit.JSON, kit.CreateMessageFactory(&wideIntMsg{}))
	bag := httpmux.Params{
		{Key: "int", Value: "3000000000"},
		{Key: "intPtr", Value: "-3000000000"},
		{Key: "uint", Value: "3000000000"},
		{Key: "u32", Value: "4000000000"},
		{Key: "u64", Value: "18446744073709551615"},
		{Key: "u64Ptr", Value: "-1"},
	}

	msg, err := dec(nil, bag, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := msg.(*wideIntMsg) //nolint:forcetypeassert
	if m.Int != 3000000000 || m.IntPtr == nil || *m.IntPtr != -3000000000 {
		t.Fatalf("int fields truncated: %d %v", m.Int, m.IntPtr)
	}
	if m.Uint != 3000000000 || m.U32 != 4000000000 {
		t.Fatalf("uint fields truncated: %d %d", m.Uint, m.U32)
	}
	if m.U64 != math.MaxUint64 {
		t.Fatalf("uint64 field clamped: %d", m.U64)
	}
	if m.U64Ptr == nil || *m.U64Ptr != 0 {
		t.Fatalf("negative uint64 must decode as 0, got %v", m.U64Ptr)
	}
}
