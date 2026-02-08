package reflector_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/kit/utils/reflector"
)

type numericMessage struct {
	I   int
	U   uint
	I32 int32
	U32 uint32
	I64 int64
	U64 uint64
	S   string
}

type privateMessage struct {
	s string
}

func TestFieldsGettersAndErrors(t *testing.T) {
	r := reflector.New()
	msg := &numericMessage{
		I:   1,
		U:   2,
		I32: 3,
		U32: 4,
		I64: 5,
		U64: 6,
		S:   "ok",
	}

	reflector.Register(msg)
	ref := r.Load(msg)
	obj := ref.Obj()
	if v, err := obj.GetInt(msg, "I"); err != nil || v != 1 {
		t.Fatalf("unexpected GetInt: %v, %d", err, v)
	}
	if v, err := obj.GetUInt(msg, "U"); err != nil || v != 2 {
		t.Fatalf("unexpected GetUInt: %v, %d", err, v)
	}
	if v, err := obj.GetInt32(msg, "I32"); err != nil || v != 3 {
		t.Fatalf("unexpected GetInt32: %v, %d", err, v)
	}
	if v, err := obj.GetUInt32(msg, "U32"); err != nil || v != 4 {
		t.Fatalf("unexpected GetUInt32: %v, %d", err, v)
	}
	if v, err := obj.GetInt64(msg, "I64"); err != nil || v != 5 {
		t.Fatalf("unexpected GetInt64: %v, %d", err, v)
	}
	if v, err := obj.GetUInt64(msg, "U64"); err != nil || v != 6 {
		t.Fatalf("unexpected GetUInt64: %v, %d", err, v)
	}
	if v, err := obj.GetString(msg, "S"); err != nil || v != "ok" {
		t.Fatalf("unexpected GetString: %v, %q", err, v)
	}
	if obj.GetIntDefault(msg, "I", 0) != 1 {
		t.Fatal("unexpected GetIntDefault")
	}
	if obj.GetUIntDefault(msg, "U", 0) != 2 {
		t.Fatal("unexpected GetUIntDefault")
	}
	if obj.GetInt64Default(msg, "I64", 0) != 5 {
		t.Fatal("unexpected GetInt64Default")
	}
	if obj.GetUInt64Default(msg, "U64", 0) != 6 {
		t.Fatal("unexpected GetUInt64Default")
	}
	if obj.GetInt32Default(msg, "I32", 0) != 3 {
		t.Fatal("unexpected GetInt32Default")
	}
	if obj.GetUInt32Default(msg, "U32", 0) != 4 {
		t.Fatal("unexpected GetUInt32Default")
	}
	if obj.GetUIntDefault(msg, "S", 9) != 9 {
		t.Fatal("unexpected GetUIntDefault fallback")
	}

	if _, err := obj.GetInt(msg, "S"); err == nil {
		t.Fatal("expected GetInt to fail on string field")
	}

	fields := 0
	obj.WalkFields(func(_ string, _ reflector.FieldInfo) {
		fields++
	})
	if fields == 0 {
		t.Fatal("expected WalkFields to visit fields")
	}

	if !strings.Contains(obj.String(), "I") {
		t.Fatalf("unexpected Fields string: %s", obj.String())
	}

	fi := obj["I"]
	if fi.Type().Name != "I" {
		t.Fatalf("unexpected field info type: %+v", fi.Type())
	}

	if ref.Type() != reflect.TypeOf(*msg) {
		t.Fatalf("unexpected reflected type: %v", ref.Type())
	}

	if v, err := r.Get(msg, "I"); err != nil || v.(int) != 1 {
		t.Fatalf("unexpected Reflector.Get: %v, %v", err, v)
	}
	if v, err := r.GetInt(msg, "I"); err != nil || v != 1 {
		t.Fatalf("unexpected Reflector.GetInt: %v, %v", err, v)
	}

	if _, err := r.GetString(msg, "missing"); err != reflector.ErrNoField {
		t.Fatalf("expected ErrNoField, got %v", err)
	}

	if _, err := r.GetString(&privateMessage{}, "s"); err != reflector.ErrNotExported {
		t.Fatalf("expected ErrNotExported, got %v", err)
	}

	if _, err := r.GetString(123, "x"); err != reflector.ErrMessageIsNotStruct {
		t.Fatalf("expected ErrMessageIsNotStruct, got %v", err)
	}
}
