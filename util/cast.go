package util

import (
	"encoding/json"

	"github.com/goccy/go-reflect"
	"github.com/jinzhu/copier"
)

// Cast Converts any type to a given type. If conversion fails, it returns the zero value of the given type.
func Cast[T any](val any) T {
	if val, ok := val.(T); ok {
		return val
	}
	var zero T

	return zero
}

// CastJSON Converts any type to a given type based on their json representation. It partially fills
// the target in case they are not directly compatible.
func CastJSON[T any](val any) T {
	return FromJSON[T](ToJSON(val))
}

// ToJSON Converts a given value to a byte array.
func ToJSON(val any) []byte {
	return Ok(json.Marshal(val))
}

// FromJSON Converts a byte array to a given type.
func FromJSON[T any](bytes []byte) T {
	var v T
	_ = json.Unmarshal(bytes, &v)

	return v
}

// ToMap Converts any type to a map[string]interface{}.
func ToMap(s any) map[string]any {
	m := make(map[string]any)
	_ = json.Unmarshal(Ok(json.Marshal(s)), &m)

	return m
}

type TypeConverter = copier.TypeConverter

func TypeConvert[SRC, DST any](fn func(src SRC) (DST, error)) TypeConverter {
	var src SRC
	var dst DST

	return TypeConverter{
		SrcType: src,
		DstType: dst,
		Fn: func(src any) (dst any, err error) {
			return fn(src.(SRC)) //nolint:forcetypeassert
		},
	}
}

func DynCast[DST, SRC any](src SRC, mappingPair ...string) DST {
	return DynCastOption[DST, SRC](src, nil, mappingPair...)
}

func DynCastOption[DST, SRC any](
	src SRC,
	converters []TypeConverter,
	mappingPair ...string,
) DST {
	var (
		out DST
		m   map[string]string
	)
	for i := 0; i < len(mappingPair); i += 2 {
		if m == nil {
			m = make(map[string]string)
		}
		m[mappingPair[i]] = mappingPair[i+1]
	}

	err := copier.CopyWithOption(&out, src, copier.Option{
		DeepCopy:   true,
		Converters: converters,
		FieldNameMapping: []copier.FieldNameMapping{
			{
				SrcType: reflect.Indirect(reflect.ValueOf(src)).Interface(),
				DstType: out,
				Mapping: m,
			},
		},
	})
	if err != nil {
		panic(err)
	}

	return out
}
