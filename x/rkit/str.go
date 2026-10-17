package rkit

import (
	"fmt"
	"strconv"
	"strings"
)

func CloneStr(s string) string {
	return string(S2B(s))
}

func CloneBytes(b []byte) []byte {
	return []byte(B2S(b))
}

// StrToFloat64 parses s as a base-10 number. Like the other StrTo* helpers it ignores the
// parse error: malformed input yields 0 and out-of-range input is clamped to the bounds of
// the target type.
func StrToFloat64(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)

	return v
}

func StrToFloat32(s string) float32 {
	v, _ := strconv.ParseFloat(s, 32)

	return float32(v)
}

func StrToInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)

	return v
}

func StrToInt32(s string) int32 {
	v, _ := strconv.ParseInt(s, 10, 32)

	return int32(v)
}

func StrToUInt64(s string) uint64 {
	return parseUint(s, 64)
}

func StrToUInt32(s string) uint32 {
	return uint32(parseUint(s, 32))
}

func StrToInt(s string) int {
	v, _ := strconv.ParseInt(s, 10, strconv.IntSize)

	return int(v)
}

func StrToUInt(s string) uint {
	return uint(parseUint(s, strconv.IntSize))
}

// parseUint accepts a leading '+' like strconv.ParseInt does; strconv.ParseUint rejects it.
func parseUint(s string, bitSize int) uint64 {
	v, _ := strconv.ParseUint(strings.TrimPrefix(s, "+"), 10, bitSize)

	return v
}

func Int64ToStr(x int64) string {
	return strconv.FormatInt(x, 10)
}

func Int32ToStr(x int32) string {
	return strconv.FormatInt(int64(x), 10)
}

func UInt64ToStr(x uint64) string {
	return strconv.FormatUint(x, 10)
}

func UInt32ToStr(x uint32) string {
	return strconv.FormatUint(uint64(x), 10)
}

func Float64ToStr(x float64) string {
	return strconv.FormatFloat(x, 'f', -1, 64)
}

func F64ToStr(x float64) string {
	return Float64ToStr(x)
}

func Float32ToStr(x float32) string {
	return strconv.FormatFloat(float64(x), 'f', -1, 32)
}

func F32ToStr(x float32) string {
	return Float32ToStr(x)
}

func IntToStr(x int) string {
	return strconv.Itoa(x)
}

func UIntToStr(x uint) string {
	return strconv.FormatUint(uint64(x), 10)
}

func StrTruncate(s string, maxSize int) string {
	count := 0
	builder := strings.Builder{}

	for _, char := range s {
		if maxSize <= 0 {
			break
		}

		builder.WriteRune(char)

		count++
		if count >= maxSize {
			break
		}
	}

	return builder.String()
}

type StrLine struct {
	format string
	args   []any
}

func (l StrLine) String() string {
	return fmt.Sprintf(l.format, l.args...)
}

func L(format string, args ...any) StrLine {
	return StrLine{format: format, args: args}
}

func StrLines(lines ...StrLine) string {
	builder := strings.Builder{}

	for _, line := range lines {
		builder.WriteString(fmt.Sprintf(line.format, line.args...))
		builder.WriteString("\n")
	}

	return builder.String()
}
