package rkit

import "testing"

func TestMustAndCoalesce(t *testing.T) {
	if got := Must("ok", nil); got != "ok" {
		t.Fatalf("Must = %q, want %q", got, "ok")
	}

	if got := Coalesce("", "", "x"); got != "x" {
		t.Fatalf("Coalesce = %q, want %q", got, "x")
	}
}

func TestGenericHelpers(t *testing.T) {
	val := 42
	if got := PtrVal(&val); got != 42 {
		t.Fatalf("PtrVal = %d, want 42", got)
	}

	if got := PtrVal[int](nil); got != 0 {
		t.Fatalf("PtrVal(nil) = %d, want 0", got)
	}

	ptr := ValPtr(7)
	if ptr == nil || *ptr != 7 {
		t.Fatalf("ValPtr = %v, want 7", ptr)
	}

	if got := ValPtrOrNil(0); got != nil {
		t.Fatalf("ValPtrOrNil(0) = %v, want nil", got)
	}

	if got := ValPtrOrNil(9); got == nil || *got != 9 {
		t.Fatalf("ValPtrOrNil(9) = %v, want 9", got)
	}

	if got := Ok("ok", nil); got != "ok" {
		t.Fatalf("Ok = %q, want %q", got, "ok")
	}

	if got := OkOr("ok", nil, "fallback"); got != "ok" {
		t.Fatalf("OkOr = %q, want %q", got, "ok")
	}

	if got := OkOr("ok", assertErr{}, "fallback"); got != "fallback" {
		t.Fatalf("OkOr err = %q, want %q", got, "fallback")
	}

	if got := TryCast[string]("x"); got != "x" {
		t.Fatalf("TryCast = %q, want %q", got, "x")
	}

	if got := TryCast[int]("x"); got != 0 {
		t.Fatalf("TryCast mismatch = %d, want 0", got)
	}

	Assert(nil)
}

type assertErr struct{}

func (assertErr) Error() string { return "fail" }

func TestArrayHelpers(t *testing.T) {
	in := []string{"a", "b", "a"}
	set := ArrayToSet(in)
	if len(set) != 2 {
		t.Fatalf("ArrayToSet len = %d, want 2", len(set))
	}

	m := ArrayToMap(in)
	if len(m) != 3 || m[0] != "a" || m[1] != "b" {
		t.Fatalf("ArrayToMap unexpected result: %+v", m)
	}

	keys := MapKeysToArray(map[string]int{"k1": 1, "k2": 2})
	if len(keys) != 2 {
		t.Fatalf("MapKeysToArray len = %d, want 2", len(keys))
	}
}
