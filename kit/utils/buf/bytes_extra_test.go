package buf

import (
	"bytes"
	"io"
	"testing"
)

func TestBytesHelpers(t *testing.T) {
	pool := NewBytesPool(2, 16)
	bb := pool.Get(0, 3)
	if bb.Cap() != 4 {
		t.Fatalf("unexpected capacity: %d", bb.Cap())
	}

	bb.AppendByte('a')
	bb.AppendString("bc")
	if string(*bb.Bytes()) != "abc" {
		t.Fatalf("unexpected bytes: %q", string(*bb.Bytes()))
	}

	dst := make([]byte, 3)
	bb.CopyTo(dst)
	if string(dst) != "abc" {
		t.Fatalf("unexpected copy: %q", string(dst))
	}

	if string(bb.AppendTo([]byte("x"))) != "xabc" {
		t.Fatalf("unexpected append to")
	}

	bb.Fill([]byte("Z"), 1, 2)
	if string(*bb.Bytes()) != "aZc" {
		t.Fatalf("unexpected fill: %q", string(*bb.Bytes()))
	}

	bb.CopyFromWithOffset([]byte("y"), 2)
	if string(*bb.Bytes()) != "aZy" {
		t.Fatalf("unexpected copy from offset: %q", string(*bb.Bytes()))
	}

	newBytes := []byte("hello")
	bb.SetBytes(&newBytes)
	if string(*bb.Bytes()) != "hello" {
		t.Fatalf("unexpected set bytes: %q", string(*bb.Bytes()))
	}

	buf := make([]byte, 5)
	n, err := bb.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("unexpected read err: %v", err)
	}
	if n == 0 {
		t.Fatal("expected read data")
	}

	bb.Reset()
	_, _ = bb.Write([]byte("ok"))
	if !bytes.Equal(*bb.Bytes(), []byte("ok")) {
		t.Fatalf("unexpected write: %q", string(*bb.Bytes()))
	}

	bb.Release()
}

func TestBytesPoolPanics(t *testing.T) {
	pool := NewBytesPool(2, 16)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on invalid length")
		}
	}()
	_ = pool.Get(5, 4)
}

func TestNewBytesPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on invalid length")
		}
	}()
	_ = newBytes(nil, 5, 4)
}

func TestLogarithmicRangeAndCeil(t *testing.T) {
	var sizes []int
	logarithmicRange(3, 8, func(n int) {
		sizes = append(sizes, n)
	})
	if len(sizes) != 2 || sizes[0] != 4 || sizes[1] != 8 {
		t.Fatalf("unexpected sizes: %v", sizes)
	}

	if ceilToPowerOfTwo(1) != 1 || ceilToPowerOfTwo(2) != 2 || ceilToPowerOfTwo(3) != 4 {
		t.Fatalf("unexpected ceil to power of two")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on too large")
		}
	}()
	_ = ceilToPowerOfTwo(maxIntHeadBit + 1)
}
