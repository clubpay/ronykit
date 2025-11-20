//go:build !go1.19 && !go1.18

package rkit

import "unsafe"

// ByteToStr converts byte slice to a string without memory allocation.
// Note it may break if string and/or slice header will change
// in the future go versions.
func ByteToStr(bts []byte) string {
	if len(bts) == 0 {
		return ""
	}

	return unsafe.String(unsafe.SliceData(bts), len(bts))
}

// B2S is alias for ByteToStr.
func B2S(bts []byte) string {
	return ByteToStr(bts)
}

// StrToByte converts string to a byte slice without memory allocation.
// Note it may break if string and/or slice header will change
// in the future go versions.
func StrToByte(str string) (b []byte) {
	if len(str) == 0 {
		return nil
	}

	return unsafe.Slice(unsafe.StringData(str), len(str))
}

// S2B is alias for StrToByte.
func S2B(str string) []byte {
	return StrToByte(str)
}
