package desc

import (
	"fmt"
	"reflect"
)

func typ(prefix string, t reflect.Type) string {
	// we need a hacky fix to handle correctly json.RawMessage and kit.RawMessage in auto-generated code
	// of the stubs
	switch t.String() {
	case "json.RawMessage":
		return fmt.Sprintf("%s%s", prefix, "kit.JSONMessage")
	case "kit.RawMessage":
		return fmt.Sprintf("%s%s", prefix, "kit.Message")
	}

	//nolint:exhaustive
	switch t.Kind() {
	case reflect.Slice:
		prefix += "[]"

		return typ(prefix, t.Elem())
	case reflect.Array:
		prefix += fmt.Sprintf("[%d]", t.Len())

		return typ(prefix, t.Elem())
	case reflect.Ptr:
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
