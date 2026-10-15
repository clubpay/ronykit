package ee

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/clubpay/ronykit/ee/store"
)

type Op = store.Op

const (
	Eq  = store.Eq
	Neq = store.Neq
	Gt  = store.Gt
	Gte = store.Gte
	Lt  = store.Lt
	Lte = store.Lte
	In  = store.In
)

// Query filters states on their top-level fields, referenced by Go field
// name. Build it with Where and chain the rest; the zero Query matches
// everything. Execution and optimization are up to the store backend.
type Query struct {
	where   []store.Cond
	orderBy []store.Order
	limit   int
}

// Where starts a query with an equality condition.
func Where(field string, value any) Query {
	return Query{}.And(field, Eq, value)
}

func (q Query) And(field string, op Op, value any) Query {
	q.where = append(slices.Clip(q.where), store.Cond{Field: field, Op: op, Value: value})

	return q
}

func (q Query) OrderBy(field string, desc bool) Query {
	q.orderBy = append(slices.Clip(q.orderBy), store.Order{Field: field, Desc: desc})

	return q
}

func (q Query) Limit(n int) Query {
	q.limit = n

	return q
}

// compile validates the query against s and lowers it to a store query:
// field names become encoded names and values are converted to the field's
// Go type, so ee.Where("Amount", 100) works against a uint64 field.
func (q Query) compile(s *stateSchema) (store.Query, error) {
	out := store.Query{Kind: s.name, Limit: q.limit}

	for _, c := range q.where {
		f, ok := s.fields[c.Field]
		if !ok {
			return store.Query{}, fmt.Errorf(
				"%w: %s has no queryable field %q",
				ErrInvalidQuery,
				s.name,
				c.Field,
			)
		}

		v, err := convertCondValue(f.typ, c.Op, c.Value)
		if err != nil {
			return store.Query{}, fmt.Errorf("%w: %s.%s: %w", ErrInvalidQuery, s.name, c.Field, err)
		}

		out.Where = append(out.Where, store.Cond{Field: f.encoded, Op: c.Op, Value: v})
	}

	for _, o := range q.orderBy {
		f, ok := s.fields[o.Field]
		if !ok {
			return store.Query{}, fmt.Errorf(
				"%w: %s has no queryable field %q",
				ErrInvalidQuery,
				s.name,
				o.Field,
			)
		}

		out.OrderBy = append(out.OrderBy, store.Order{Field: f.encoded, Desc: o.Desc})
	}

	return out, nil
}

func convertCondValue(t reflect.Type, op Op, v any) (any, error) {
	if op != In {
		return convertValue(t, v)
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("operator %s expects a slice, got %T", op, v)
	}

	out := make([]any, rv.Len())
	for i := range out {
		x, err := convertValue(t, rv.Index(i).Interface())
		if err != nil {
			return nil, err
		}

		out[i] = x
	}

	return out, nil
}

func convertValue(t reflect.Type, v any) (any, error) {
	rv := reflect.ValueOf(v)

	switch {
	case !rv.IsValid():
		return nil, fmt.Errorf("nil value for %s field", t)
	case rv.Type() == t:
		return v, nil
	case isNumber(rv.Kind()) && isNumber(t.Kind()):
		out := rv.Convert(t)
		// Reject lossy conversions: a round trip catches truncation and
		// overflow, but not a sign flip, which wraps both ways.
		if !out.Convert(rv.Type()).Equal(rv) || isNegative(rv) != isNegative(out) {
			return nil, fmt.Errorf("value %v does not fit in %s", v, t)
		}

		return out.Interface(), nil
	case rv.Type().AssignableTo(t):
		return rv.Convert(t).Interface(), nil
	default:
		return nil, fmt.Errorf("value of type %T cannot be compared to %s", v, t)
	}
}

func isNumber(k reflect.Kind) bool {
	return (k >= reflect.Int && k <= reflect.Uint64) || k == reflect.Float32 || k == reflect.Float64
}

func isNegative(v reflect.Value) bool {
	switch {
	case v.Kind() >= reflect.Int && v.Kind() <= reflect.Int64:
		return v.Int() < 0
	case v.Kind() == reflect.Float32 || v.Kind() == reflect.Float64:
		return v.Float() < 0
	default:
		return false
	}
}
