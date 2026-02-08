package desc

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/clubpay/ronykit/kit"
)

type stringerType struct{}

func (stringerType) String() string { return "x" }

func TestParseKind(t *testing.T) {
	if parseKind(reflect.TypeOf(true)) != Bool {
		t.Fatalf("unexpected kind for bool")
	}
	if parseKind(reflect.TypeOf("")) != String {
		t.Fatalf("unexpected kind for string")
	}
	if parseKind(reflect.TypeOf(int(0))) != Integer {
		t.Fatalf("unexpected kind for int")
	}
	if parseKind(reflect.TypeOf(float32(0))) != Float {
		t.Fatalf("unexpected kind for float")
	}
	if parseKind(reflect.TypeOf(map[string]int{})) != Map {
		t.Fatalf("unexpected kind for map")
	}
	if parseKind(reflect.TypeOf([]int{})) != Array {
		t.Fatalf("unexpected kind for slice")
	}
	if parseKind(reflect.TypeOf(kit.RawMessage{})) != KitRawMessage {
		t.Fatalf("unexpected kind for raw message")
	}
	if parseKind(reflect.TypeOf(kit.MultipartFormMessage{})) != KitMultipartFormMessage {
		t.Fatalf("unexpected kind for multipart message")
	}
	if parseKind(reflect.TypeOf(stringerType{})) != String {
		t.Fatalf("unexpected kind for stringer")
	}
	if parseKind(reflect.TypeFor[fmt.Stringer]()) != Object {
		t.Fatalf("unexpected kind for interface")
	}
}

func TestGetParsedStructTag(t *testing.T) {
	tag := reflect.StructTag(`json:"name,omitempty" swag:"optional;deprecated;enum:a,b"`)
	pst := getParsedStructTag(tag, "json")
	if pst.Value != "name" || !pst.OmitEmpty || !pst.Optional || !pst.Deprecated {
		t.Fatalf("unexpected parsed struct tag: %+v", pst)
	}
	if len(pst.PossibleValues) != 2 || pst.PossibleValues[0] != "a" || pst.PossibleValues[1] != "b" {
		t.Fatalf("unexpected enum values: %v", pst.PossibleValues)
	}

	tag = reflect.StructTag(`json:"other,omitzero"`)
	pst = getParsedStructTag(tag, "json")
	if !pst.OmitZero || pst.Value != "other" {
		t.Fatalf("unexpected omitzero parsing: %+v", pst)
	}
}
