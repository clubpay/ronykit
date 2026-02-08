package rkit

import "testing"

func TestCloneAndConversions(t *testing.T) {
	if got := CloneStr("abc"); got != "abc" {
		t.Fatalf("CloneStr = %q, want %q", got, "abc")
	}
	if got := CloneBytes([]byte("xyz")); string(got) != "xyz" {
		t.Fatalf("CloneBytes = %q, want %q", string(got), "xyz")
	}

	if got := StrToInt("12"); got != 12 {
		t.Fatalf("StrToInt = %d, want 12", got)
	}
	if got := StrToUInt("12"); got != 12 {
		t.Fatalf("StrToUInt = %d, want 12", got)
	}
	if got := StrToInt64("12"); got != 12 {
		t.Fatalf("StrToInt64 = %d, want 12", got)
	}
	if got := StrToUInt64("12"); got != 12 {
		t.Fatalf("StrToUInt64 = %d, want 12", got)
	}
	if got := StrToInt32("12"); got != 12 {
		t.Fatalf("StrToInt32 = %d, want 12", got)
	}
	if got := StrToUInt32("12"); got != 12 {
		t.Fatalf("StrToUInt32 = %d, want 12", got)
	}
	if got := StrToFloat64("1.25"); got != 1.25 {
		t.Fatalf("StrToFloat64 = %v, want 1.25", got)
	}
	if got := StrToFloat32("1.25"); got != 1.25 {
		t.Fatalf("StrToFloat32 = %v, want 1.25", got)
	}

	if got := StrToInt("nope"); got != 0 {
		t.Fatalf("StrToInt invalid = %d, want 0", got)
	}
}

func TestNumberToString(t *testing.T) {
	if got := IntToStr(42); got != "42" {
		t.Fatalf("IntToStr = %q, want %q", got, "42")
	}
	if got := UIntToStr(42); got != "42" {
		t.Fatalf("UIntToStr = %q, want %q", got, "42")
	}
	if got := Int64ToStr(42); got != "42" {
		t.Fatalf("Int64ToStr = %q, want %q", got, "42")
	}
	if got := UInt64ToStr(42); got != "42" {
		t.Fatalf("UInt64ToStr = %q, want %q", got, "42")
	}
	if got := Int32ToStr(42); got != "42" {
		t.Fatalf("Int32ToStr = %q, want %q", got, "42")
	}
	if got := UInt32ToStr(42); got != "42" {
		t.Fatalf("UInt32ToStr = %q, want %q", got, "42")
	}
	if got := Float64ToStr(1.5); got != "1.5" {
		t.Fatalf("Float64ToStr = %q, want %q", got, "1.5")
	}
	if got := Float32ToStr(1.5); got != "1.5" {
		t.Fatalf("Float32ToStr = %q, want %q", got, "1.5")
	}
	if got := F64ToStr(2.25); got != "2.25" {
		t.Fatalf("F64ToStr = %q, want %q", got, "2.25")
	}
	if got := F32ToStr(2.25); got != "2.25" {
		t.Fatalf("F32ToStr = %q, want %q", got, "2.25")
	}
}

func TestStrTruncate(t *testing.T) {
	if got := StrTruncate("hello", 2); got != "he" {
		t.Fatalf("StrTruncate = %q, want %q", got, "he")
	}
	if got := StrTruncate("hello", 0); got != "" {
		t.Fatalf("StrTruncate zero = %q, want %q", got, "")
	}
	if got := StrTruncate("hello", 10); got != "hello" {
		t.Fatalf("StrTruncate large = %q, want %q", got, "hello")
	}
}
