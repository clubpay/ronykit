package utils

import (
	"encoding/json"
	"strconv"
	"strings"
)

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
	return strconv.FormatUint(uint64(x), 10)
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
		builder.WriteString(string(char))

		count++
		if count >= maxSize {
			break
		}
	}

	return builder.String()
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
