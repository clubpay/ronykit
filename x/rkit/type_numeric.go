package rkit

import (
	"encoding/json"
	"strconv"
)

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

	err := json.Unmarshal(bb, m)
	if err != nil {
		return err
	}

	*n = ParseNumeric(*m)

	return nil
}

func (n *Numeric) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.String())
}

func (n *Numeric) Value() float64 {
	return n.value
}

func (n *Numeric) String() string {
	if n.precision == 0 {
		n.precision = defaultPrecision
	}

	return strconv.FormatFloat(n.Value(), 'f', n.precision, 64)
}

func (n *Numeric) WithPrecision(p int) *Numeric {
	n.precision = p

	return n
}

func (n *Numeric) WithoutPrecision() *Numeric {
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
