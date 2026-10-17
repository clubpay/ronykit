package unsafeconv

import "unsafe"

func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b)) // want `hand-rolled zero-copy \[\]byte→string conversion: use rkit.B2S`
}

func s2b(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s)) // want `hand-rolled zero-copy string→\[\]byte conversion: use rkit.S2B`
}

func headerB2S(b []byte) string {
	return *(*string)(unsafe.Pointer(&b)) // want `use rkit.B2S`
}

func headerS2B(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s)) // want `use rkit.S2B`
}

// Not reported below.

func mismatched(a, b []byte) string {
	return unsafe.String(unsafe.SliceData(a), len(b))
}

func prefix(b []byte, n int) string {
	return unsafe.String(unsafe.SliceData(b), n)
}

func intPun(x *int64) uint64 {
	return *(*uint64)(unsafe.Pointer(x))
}
