package rkit

import (
	"strconv"
	"testing"
)

type castSrc struct {
	Name string
	Age  int
}

type castDst struct {
	Name string
}

func TestCast(t *testing.T) {
	if got := Cast[string]("ok"); got != "ok" {
		t.Fatalf("Cast = %q, want %q", got, "ok")
	}

	if got := Cast[int]("nope"); got != 0 {
		t.Fatalf("Cast mismatch = %d, want 0", got)
	}
}

func TestCastJSON(t *testing.T) {
	src := castSrc{Name: "Tom", Age: 20}
	dst := CastJSON[castDst](src)
	if dst.Name != "Tom" {
		t.Fatalf("CastJSON = %+v, want Name Tom", dst)
	}
}

func TestJSONHelpers(t *testing.T) {
	src := castSrc{Name: "Ana", Age: 30}
	bytes := ToJSON(src)
	if len(bytes) == 0 {
		t.Fatalf("ToJSON produced empty bytes")
	}

	dst := FromJSON[castSrc](bytes)
	if dst != src {
		t.Fatalf("FromJSON = %+v, want %+v", dst, src)
	}

	m := ToMap(src)
	if m["Name"] != "Ana" {
		t.Fatalf("ToMap Name = %v, want Ana", m["Name"])
	}
	if int(m["Age"].(float64)) != 30 { // json unmarshal uses float64 for numbers
		t.Fatalf("ToMap Age = %v, want 30", m["Age"])
	}
}

func TestDynCastWithMapping(t *testing.T) {
	type src struct {
		First string
		Age   int
	}
	type dst struct {
		Name string
		Age  int
	}

	out := DynCast[dst](src{First: "Sam", Age: 28}, "First", "Name")
	if out.Name != "Sam" || out.Age != 28 {
		t.Fatalf("DynCast = %+v, want Name Sam Age 28", out)
	}
}

func TestDynCastWithConverter(t *testing.T) {
	type src struct {
		Age int
	}
	type dst struct {
		Age string
	}

	converters := []TypeConverter{
		TypeConvert(func(src int) (string, error) {
			return strconv.Itoa(src), nil
		}),
	}

	out := DynCastOption[dst](src{Age: 12}, converters)
	if out.Age != "12" {
		t.Fatalf("DynCastOption Age = %q, want %q", out.Age, "12")
	}
}
