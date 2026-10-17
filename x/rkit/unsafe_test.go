package rkit

import "testing"

func TestByteToStr(t *testing.T) {
	b := []byte("abc")
	s := ByteToStr(b)
	if s != "abc" {
		t.Fatalf("ByteToStr = %q, want %q", s, "abc")
	}

	b[0] = 'x'
	if s != "xbc" {
		t.Fatalf("ByteToStr reflects change = %q, want %q", s, "xbc")
	}
}

func TestStrToByte(t *testing.T) {
	s := "abc"
	b := StrToByte(s)
	if string(b) != "abc" {
		t.Fatalf("StrToByte = %q, want %q", string(b), "abc")
	}
	if len(b) != len(s) {
		t.Fatalf("StrToByte len = %d, want %d", len(b), len(s))
	}
}

func TestUnsafeConversionsEmpty(t *testing.T) {
	if got := ByteToStr(nil); got != "" {
		t.Fatalf("ByteToStr(nil) = %q, want empty", got)
	}
	if got := ByteToStr([]byte{}); got != "" {
		t.Fatalf("ByteToStr(empty) = %q, want empty", got)
	}
	if got := StrToByte(""); got != nil {
		t.Fatalf("StrToByte(\"\") = %v, want nil", got)
	}
}
