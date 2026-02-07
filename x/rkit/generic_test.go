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
