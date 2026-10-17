package rkit

import "unsafe"

// ByteToStr converts byte slice to a string without memory allocation.
// The string shares memory with bts, so bts must not be modified while the string is in use.
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
// The slice shares memory with str and must never be modified. It returns nil for an empty string.
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
