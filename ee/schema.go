package ee

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	tagName = "ee"
	// reservedPrefix is kept for kinds ee itself stores (e.g. DedupKind).
	reservedPrefix = "ee."
)

// State is implemented by every state type. StateName must be a constant of
// the type (it is called on the zero value) and must use a value receiver.
// Versioning is done with a new type and a new name, e.g. "user.v2".
type State interface {
	StateName() string
}

// StateSchemaInfo describes a registered state. Backends may use it in their
// bootstrap/migration helpers.
type StateSchemaInfo struct {
	Name string
	Type reflect.Type
}

type stateSchema struct {
	name    string
	typ     reflect.Type
	idIndex []int
	// fields maps queryable Go field names to their encoded names and types.
	fields map[string]stateField
}

type stateField struct {
	encoded string
	typ     reflect.Type
}

func parseStateSchema(name string, t reflect.Type, codec Codec) (*stateSchema, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: %s: empty StateName", ErrInvalidSchema, t)
	}

	if strings.HasPrefix(name, reservedPrefix) {
		return nil, fmt.Errorf(
			"%w: %s: StateName %q uses the reserved %q prefix",
			ErrInvalidSchema,
			t,
			name,
			reservedPrefix,
		)
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: %s: state must be a struct", ErrInvalidSchema, t)
	}

	s := &stateSchema{name: name, typ: t, fields: map[string]stateField{}}
	for f := range t.Fields() {
		if enc, ok := codec.FieldName(f); ok {
			s.fields[f.Name] = stateField{encoded: enc, typ: f.Type}
		}

		opts := tagOptions(f)
		if len(opts) == 0 {
			continue
		}

		for _, o := range opts {
			if o != "id" {
				return nil, fmt.Errorf("%w: %s.%s: unknown tag option %q", ErrInvalidSchema, t, f.Name, o)
			}

			if s.idIndex != nil {
				return nil, fmt.Errorf("%w: %s: more than one id field", ErrInvalidSchema, t)
			}

			if !f.IsExported() || f.Type.Kind() != reflect.String {
				return nil, fmt.Errorf(
					"%w: %s.%s: id must be an exported string field",
					ErrInvalidSchema,
					t,
					f.Name,
				)
			}

			s.idIndex = f.Index
		}
	}

	if s.idIndex == nil {
		return nil, fmt.Errorf("%w: %s: missing `ee:\"id\"` field", ErrInvalidSchema, t)
	}

	return s, nil
}

func (s *stateSchema) info() StateSchemaInfo {
	return StateSchemaInfo{Name: s.name, Type: s.typ}
}

// id reads the id of v, which must be a value of s.typ.
func (s *stateSchema) id(v reflect.Value) string {
	return v.FieldByIndex(s.idIndex).String()
}

func tagOptions(f reflect.StructField) []string {
	tag, ok := f.Tag.Lookup(tagName)
	if !ok || tag == "" {
		return nil
	}

	return strings.Split(tag, ",")
}
