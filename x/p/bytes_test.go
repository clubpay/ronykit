package p

import (
	"bytes"
	"testing"
)

func TestBytesPoolGetAndRelease(t *testing.T) {
	pool := NewBytesPool(2, 32)
	b := pool.Get(2, 8)
	if b.Len() != 2 {
		t.Fatalf("expected len 2, got %d", b.Len())
	}
	if b.Cap() < 8 {
		t.Fatalf("expected cap at least 8, got %d", b.Cap())
	}

	b.CopyFrom([]byte("hi"))
	out := make([]byte, 2)
	b.CopyTo(out)
	if !bytes.Equal(out, []byte("hi")) {
		t.Fatalf("unexpected copy result: %q", out)
	}

	b.Reset()
	if b.Len() != 0 {
		t.Fatalf("expected reset len 0, got %d", b.Len())
	}
	b.Release()
}

func TestCeilToPowerOfTwo(t *testing.T) {
	tests := map[int]int{
		1: 1,
		2: 2,
		3: 4,
		8: 8,
		9: 16,
	}
	for in, want := range tests {
		if got := ceilToPowerOfTwo(in); got != want {
			t.Fatalf("ceilToPowerOfTwo(%d) = %d, want %d", in, got, want)
		}
	}
}
