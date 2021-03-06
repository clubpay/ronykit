package desc

import (
	"fmt"
	"reflect"
)

func typ(prefix string, t reflect.Type) string {
	//nolint:exhaustive
	switch t.Kind() {
	case reflect.Slice:
		prefix += "[]"
	case reflect.Array:
		prefix += fmt.Sprintf("[%d]", t.Len())
	case reflect.Ptr:
		prefix += "*"
	case reflect.Interface:
		in := t.Name()
		if in == "" {
			in = "interface{}"
		}

		return fmt.Sprintf("%s%s", prefix, in)
	case reflect.Struct:
		return fmt.Sprintf("%s%s", prefix, t.Name())
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", typ("", t.Key()), typ("", t.Elem()))
	default:
		return fmt.Sprintf("%s%s", prefix, t.Kind().String())
	}

	return typ(prefix, t.Elem())
}
