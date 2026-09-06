package desc

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/clubpay/ronykit/kit"
)

func typ(prefix string, t reflect.Type) string {
	// we need a hacky fix to handle correctly json.RawMessage and kit.RawMessage in auto-generated code
	// of the stubs.
	// NOTE: compare reflect.Type directly instead of Type.String(); since Go 1.27
	// json.RawMessage is an alias for jsontext.Value and its String() no longer
	// reports "json.RawMessage".
	switch t {
	case reflect.TypeFor[json.RawMessage]():
		return fmt.Sprintf("%s%s", prefix, "kit.JSONMessage")
	case reflect.TypeFor[kit.RawMessage]():
		return fmt.Sprintf("%s%s", prefix, "kit.RawMessage")
	case reflect.TypeFor[kit.MultipartFormMessage]():
		return fmt.Sprintf("%s%s", prefix, "kit.MultipartFormMessage")
	}

	//nolint:exhaustive
	switch t.Kind() {
	case reflect.Slice:
		prefix += "[]"

		return typ(prefix, t.Elem())
	case reflect.Array:
		prefix += fmt.Sprintf("[%d]", t.Len())

		return typ(prefix, t.Elem())
	case reflect.Pointer:
		prefix += "*"

		return typ(prefix, t.Elem())
	case reflect.Interface:
		in := t.Name()
		if in == "" {
			in = "any"
		}

		return fmt.Sprintf("%s%s", prefix, in)
	case reflect.Struct:
		return fmt.Sprintf("%s%s", prefix, t.Name())
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", typ("", t.Key()), typ("", t.Elem()))
	default:
		return fmt.Sprintf("%s%s", prefix, t.Kind().String())
	}
}
