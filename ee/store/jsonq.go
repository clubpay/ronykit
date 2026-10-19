package store

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/clubpay/ronykit/x/rkit"
)

// DecodeJSON reads the top-level fields of a JSON object. Numbers are
// json.Number. It matches ee.JSONCodec.
func DecodeJSON(data []byte) (map[string]any, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()

	var out map[string]any
	if err := d.Decode(&out); err != nil {
		return nil, err
	}

	return out, nil
}

// MatchAll reports whether doc satisfies every condition. doc values come
// from DecodeJSON; condition values have the Go type of the entity field.
func MatchAll(doc map[string]any, conds []Cond) bool {
	for _, c := range conds {
		if !match(doc[c.Field], c) {
			return false
		}
	}

	return true
}

func match(v any, c Cond) bool {
	if c.Op == In {
		vals := rkit.TryCast[[]any](c.Value)
		for _, x := range vals {
			if r, ok := compareValue(v, x); ok && r == 0 {
				return true
			}
		}

		return false
	}

	r, ok := compareValue(v, c.Value)
	if !ok {
		return false
	}

	switch c.Op {
	case Eq:
		return r == 0
	case Neq:
		return r != 0
	case Gt:
		return r > 0
	case Gte:
		return r >= 0
	case Lt:
		return r < 0
	case Lte:
		return r <= 0
	default:
		return false
	}
}

// compareValue compares a decoded document value with a condition value.
// ok is false when they are not comparable.
func compareValue(doc, want any) (int, bool) {
	if t, ok := want.(time.Time); ok {
		d, ok := toTime(doc)

		return d.Compare(t), ok
	}

	rv := reflect.ValueOf(want)

	switch {
	case rv.Kind() == reflect.String:
		d, ok := doc.(string)

		return strings.Compare(d, rv.String()), ok
	case rv.Kind() == reflect.Bool:
		d, ok := doc.(bool)

		return compareBool(d, rv.Bool()), ok
	case rv.CanInt():
		d, ok := toNumber(doc)
		if !ok {
			return 0, false
		}

		i, err := strconv.ParseInt(d, 10, 64)
		if err != nil {
			return compareFloat(d, float64(rv.Int()))
		}

		return cmp.Compare(i, rv.Int()), true
	case rv.CanUint():
		d, ok := toNumber(doc)
		if !ok {
			return 0, false
		}

		u, err := strconv.ParseUint(d, 10, 64)
		if err != nil {
			return compareFloat(d, float64(rv.Uint()))
		}

		return cmp.Compare(u, rv.Uint()), true
	case rv.CanFloat():
		d, ok := toNumber(doc)
		if !ok {
			return 0, false
		}

		return compareFloat(d, rv.Float())
	default:
		return 0, false
	}
}

// CompareValues orders two decoded document values for Query.OrderBy.
// Missing or mixed values sort first.
func CompareValues(a, b any) int {
	if an, ok := toNumber(a); ok {
		if bn, ok := toNumber(b); ok {
			af := rkit.StrToFloat64(an)
			bf := rkit.StrToFloat64(bn)

			return cmp.Compare(af, bf)
		}
	}

	switch a := a.(type) {
	case string:
		if b, ok := b.(string); ok {
			return strings.Compare(a, b)
		}
	case bool:
		if b, ok := b.(bool); ok {
			return compareBool(a, b)
		}
	}

	return cmp.Compare(rank(a), rank(b))
}

func rank(v any) int {
	if v == nil {
		return 0
	}

	return 1
}

func toNumber(v any) (string, bool) {
	switch v := v.(type) {
	case json.Number:
		return v.String(), true
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(v), true
	default:
		return "", false
	}
}

func toTime(v any) (time.Time, bool) {
	switch v := v.(type) {
	case time.Time:
		return v, true
	case string:
		t, err := time.Parse(time.RFC3339Nano, v)

		return t, err == nil
	default:
		return time.Time{}, false
	}
}

func compareFloat(doc string, want float64) (int, bool) {
	f, err := strconv.ParseFloat(doc, 64)

	return cmp.Compare(f, want), err == nil
}

func compareBool(a, b bool) int {
	switch {
	case a == b:
		return 0
	case !a:
		return -1
	default:
		return 1
	}
}
