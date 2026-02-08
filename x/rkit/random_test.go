package rkit

import "testing"

func TestRandomID(t *testing.T) {
	id := RandomID(16)
	if len(id) != 16 {
		t.Fatalf("RandomID len = %d, want 16", len(id))
	}
	for i := 0; i < len(id); i++ {
		if !containsByte(alphaNumerics, id[i]) {
			t.Fatalf("RandomID char = %q not alphanumeric", id[i])
		}
	}
}

func TestRandomDigit(t *testing.T) {
	id := RandomDigit(12)
	if len(id) != 12 {
		t.Fatalf("RandomDigit len = %d, want 12", len(id))
	}
	for i := 0; i < len(id); i++ {
		if !containsByte(digits, id[i]) {
			t.Fatalf("RandomDigit char = %q not digit", id[i])
		}
	}
}

func TestRandomIDs(t *testing.T) {
	ids := RandomIDs(1, 2, 3)
	if len(ids) != 3 {
		t.Fatalf("RandomIDs len = %d, want 3", len(ids))
	}
	if len(ids[0]) != 1 || len(ids[1]) != 2 || len(ids[2]) != 3 {
		t.Fatalf("RandomIDs lengths = %v, want [1 2 3]", []int{len(ids[0]), len(ids[1]), len(ids[2])})
	}
}

func TestRandomRanges(t *testing.T) {
	for i := 0; i < 10; i++ {
		if got := RandomInt64(10); got < 0 || got >= 10 {
			t.Fatalf("RandomInt64 = %d, want [0,10)", got)
		}
		if got := RandomInt32(10); got < 0 || got >= 10 {
			t.Fatalf("RandomInt32 = %d, want [0,10)", got)
		}
		if got := RandomInt(10); got < 0 || got >= 10 {
			t.Fatalf("RandomInt = %d, want [0,10)", got)
		}
		if got := RandomUint64(10); got >= 10 {
			t.Fatalf("RandomUint64 = %d, want [0,10)", got)
		}
		if got := SecureRandomInt63(10); got <= -10 || got >= 10 {
			t.Fatalf("SecureRandomInt63 = %d, want [0,10)", got)
		}
	}
}

func containsByte(set string, b byte) bool {
	for i := 0; i < len(set); i++ {
		if set[i] == b {
			return true
		}
	}

	return false
}
