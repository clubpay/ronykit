package util

import (
	"encoding/binary"
	"strings"
)

func CloneStr(s string) string {
	return string(S2B(s))
}

func CloneBytes(b []byte) []byte {
	return []byte(B2S(b))
}

/*
	Strings Builder helper functions
*/

func AppendStrInt(sb *strings.Builder, x int) {
	var b [8]byte

	binary.BigEndian.PutUint64(b[:], uint64(x))
	sb.Write(b[:])
}

func AppendStrUInt(sb *strings.Builder, x uint) {
	var b [8]byte

	binary.BigEndian.PutUint64(b[:], uint64(x))
	sb.Write(b[:])
}

func AppendStrInt64(sb *strings.Builder, x int64) {
	var b [8]byte

	binary.BigEndian.PutUint64(b[:], uint64(x))
	sb.Write(b[:])
}

func AppendStrUInt64(sb *strings.Builder, x uint64) {
	var b [8]byte

	binary.BigEndian.PutUint64(b[:], x)
	sb.Write(b[:])
}

func AppendStrInt32(sb *strings.Builder, x int32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(x))
	sb.Write(b[:])
}

func AppendStrUInt32(sb *strings.Builder, x uint32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], x)
	sb.Write(b[:])
}
