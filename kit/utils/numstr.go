package utils

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

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

/*
	String Conversion helper functions
*/

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
	v, _ := strconv.ParseInt(s, 10, 64)

	return uint64(v)
}

func StrToUInt32(s string) uint32 {
	v, _ := strconv.ParseInt(s, 10, 32)

	return uint32(v)
}

func StrToInt(s string) int {
	v, _ := strconv.ParseInt(s, 10, 32)

	return int(v)
}

func StrToUInt(s string) uint {
	v, _ := strconv.ParseInt(s, 10, 32)

	return uint(v)
}

func Int64ToStr(x int64) string {
	return strconv.FormatInt(x, 10)
}

func Int64ToBigEndian(x int64) [8]byte {
	var b [8]byte

	binary.BigEndian.PutUint64(b[:], uint64(x))

	return b
}

func Int32ToStr(x int32) string {
	return strconv.FormatInt(int64(x), 10)
}

func Int32ToBigEndian(x int32) [4]byte {
	var b [4]byte

	binary.BigEndian.PutUint32(b[:], uint32(x))

	return b
}

func UInt64ToStr(x uint64) string {
	return strconv.FormatUint(x, 10)
}

func UInt64ToBigEndian(x uint64) [8]byte {
	var b [8]byte

	binary.BigEndian.PutUint64(b[:], x)

	return b
}

func UInt32ToStr(x uint32) string {
	return strconv.FormatUint(uint64(x), 10)
}

func UInt32ToBigEndian(x uint32) [4]byte {
	var b [4]byte

	binary.BigEndian.PutUint32(b[:], x)

	return b
}

func Float64ToStr(x float64) string {
	return strconv.FormatFloat(x, 'f', -1, 64)
}

func Float32ToStr(x float32) string {
	return strconv.FormatFloat(float64(x), 'f', -1, 32)
}

func IntToStr(x int) string {
	return strconv.FormatUint(uint64(x), 10)
}

func StrTruncate(s string, maxSize int) string {
	count := 0
	builder := strings.Builder{}
	for _, char := range s {
		if maxSize <= 0 {
			break
		}
		builder.WriteString(string(char))

		count++
		if count >= maxSize {
			break
		}
	}

	return builder.String()
}

// ByteToStr converts byte slice to a string without memory allocation.
// Note it may break if string and/or slice header will change
// in the future go versions.
func ByteToStr(bts []byte) string {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&bts))

	var s string
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	sh.Data = bh.Data
	sh.Len = bh.Len

	return s
}

// B2S is alias for ByteToStr.
func B2S(bts []byte) string {
	return ByteToStr(bts)
}

// StrToByte converts string to a byte slice without memory allocation.
// Note it may break if string and/or slice header will change
// in the future go versions.
func StrToByte(str string) (b []byte) {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := (*reflect.StringHeader)(unsafe.Pointer(&str))
	bh.Data = sh.Data
	bh.Len = sh.Len
	bh.Cap = sh.Len

	return b
}

// S2B is alias for StrToByte.
func S2B(str string) []byte {
	return StrToByte(str)
}

func CloneStr(s string) string {
	return string(S2B(s))
}

func CloneBytes(b []byte) []byte {
	return []byte(B2S(b))
}

// Numeric represents float64 number which is decodable from string, int or float.
// It's useful when a struct field should be numeric but form of the data being decoded from is unknown or variable.
type Numeric struct {
	value     float64
	precision int
}

const defaultPrecision = 2

func (n *Numeric) UnmarshalJSON(bb []byte) error {
	type medium any
	m := new(medium)
	if err := json.Unmarshal(bb, m); err != nil {
		return err
	}

	*n = ParseNumeric(*m)

	return nil
}

func (n Numeric) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.String())
}

func (n Numeric) Value() float64 {
	return n.value
}

func (n Numeric) String() string {
	if n.precision == 0 {
		n.precision = defaultPrecision
	}

	return strconv.FormatFloat(n.Value(), 'f', n.precision, 64)
}

func (n Numeric) WithPrecision(p int) Numeric {
	n.precision = p

	return n
}

func (n Numeric) WithoutPrecision() Numeric {
	return n.WithPrecision(-1)
}

// ParseNumeric converts int, string, float to Numeric.
func ParseNumeric(src any) Numeric {
	var number float64
	switch v := src.(type) {
	case float64:
		number = v

	case string:
		number, _ = strconv.ParseFloat(v, 64)

	case int64:
		number = float64(v)

	case float32:
		return ParseNumeric(float64(v))

	case int:
		return ParseNumeric(int64(v))
	}

	return Numeric{
		value:     number,
		precision: defaultPrecision,
	}
}
