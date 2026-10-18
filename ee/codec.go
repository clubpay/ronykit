package ee

import (
	"encoding/json"
	"reflect"
	"strings"
)

// Codec encodes entities and event payloads. It must be the same on every
// replica sharing a bus or store.
type Codec interface {
	Marshal(v any) ([]byte, error)
	Unmarshal(data []byte, v any) error
	// FieldName is the encoded name of a struct field, used to translate
	// query conditions for the store. ok is false for fields the codec skips.
	FieldName(f reflect.StructField) (name string, ok bool)
}

type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error)      { return json.Marshal(v) }
func (jsonCodec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

func (jsonCodec) FieldName(f reflect.StructField) (string, bool) {
	if !f.IsExported() {
		return "", false
	}

	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	switch name {
	case "-":
		return "", false
	case "":
		return f.Name, true
	default:
		return name, true
	}
}

// JSONCodec is the default codec.
var JSONCodec Codec = jsonCodec{}
