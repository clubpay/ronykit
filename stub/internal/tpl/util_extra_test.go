package tpl

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/clubpay/ronykit/kit"
)

type testStruct struct {
	Name string
}

func (testStruct) String() string { return "x" }

func TestGoTypeHelpers(t *testing.T) {
	if goType(reflect.TypeOf(kit.RawMessage{})) != "kit.RawMessage" {
		t.Fatalf("unexpected raw message type")
	}
	if goType(reflect.TypeOf(json.RawMessage{})) != "kit.JSONMessage" {
		t.Fatalf("unexpected json message type")
	}
	if goType(reflect.TypeOf(time.Time{})) != "time.Time" {
		t.Fatalf("unexpected time type")
	}
	if goType(reflect.TypeOf([]string{})) != "[]string" {
		t.Fatalf("unexpected slice type")
	}
	if goType(reflect.TypeOf([2]int{})) != "[2]int" {
		t.Fatalf("unexpected array type")
	}
	if goType(reflect.TypeOf((*io.Reader)(nil)).Elem()) != "Reader" {
		t.Fatalf("unexpected interface type")
	}
	if goType(reflect.TypeOf(map[string]int{})) != "map[string]int" {
		t.Fatalf("unexpected map type")
	}
	if goType(reflect.TypeOf(&testStruct{})) != "*testStruct" {
		t.Fatalf("unexpected pointer type")
	}
	if goType(reflect.TypeOf(true)) != "bool" {
		t.Fatalf("unexpected bool type")
	}
}

func TestTsTypeHelpers(t *testing.T) {
	if tsType(reflect.TypeOf(time.Time{})) != "string" {
		t.Fatalf("unexpected time ts type")
	}
	if tsType(reflect.TypeOf([]byte{})) != "string" {
		t.Fatalf("unexpected []byte ts type")
	}
	if tsType(reflect.TypeOf([2]byte{})) != "string" {
		t.Fatalf("unexpected [2]byte ts type")
	}
	if tsType(reflect.TypeOf(json.RawMessage{})) != "any" {
		t.Fatalf("unexpected json raw ts type")
	}
	if tsType(reflect.TypeOf(map[string]int{})) != "{[key: string]: number}" {
		t.Fatalf("unexpected map ts type")
	}
	if tsType(reflect.TypeOf((*testStruct)(nil))) != "testStruct" {
		t.Fatalf("unexpected pointer struct ts type")
	}
	if tsType(reflect.TypeOf((*interface{})(nil)).Elem()) != "any" {
		t.Fatalf("unexpected interface ts type")
	}
	if tsType(reflect.TypeOf(true)) != "boolean" {
		t.Fatalf("unexpected bool ts type")
	}
}

func TestStringHelpers(t *testing.T) {
	if tsReplacePathParams("/v1/{id}", "req.") != "/v1/${req.id}" {
		t.Fatalf("unexpected path param replace")
	}

	arr := strAppend([]string{"a"}, "b")
	if len(arr) != 2 || arr[1] != "b" {
		t.Fatalf("unexpected strAppend result: %v", arr)
	}
	if len(strEmptySlice()) != 0 {
		t.Fatalf("unexpected empty slice")
	}

	quoted := FuncMaps["strQuote"].(func([]string) []string)([]string{"a", "b"})
	if quoted[0] != `"a"` {
		t.Fatalf("unexpected strQuote: %v", quoted)
	}
}

func TestTemplateFuncs(t *testing.T) {
	tmpl := template.Must(template.New("x").Funcs(FuncMaps).Parse("{{camelCase .}}"))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, "hello_world"); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "HelloWorld" {
		t.Fatalf("unexpected template output: %s", buf.String())
	}
}
