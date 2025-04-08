package tpl

import (
	"fmt"
	"go/build"
	"path"
	"reflect"
	"regexp"
	"strings"

	"github.com/ronaksoft/ronykit/kit"
)

func goType(t reflect.Type) string {
	return goTypeRecursive("", t)
}

func goTypeRecursive(prefix string, t reflect.Type) string {
	// we need a hacky fix to correctly handle json.RawMessage and kit.RawMessage in auto-generated code
	// of the stubs
	if t.PkgPath() == reflect.TypeOf(kit.RawMessage{}).PkgPath() {
		return fmt.Sprintf("%s%s%s", prefix, "kit.", t.Name())
	}
	switch t.String() {
	case "json.RawMessage":
		return fmt.Sprintf("%s%s", prefix, "kit.JSONMessage")
	}

	//nolint:exhaustive
	switch t.Kind() {
	case reflect.Slice:
		prefix += "[]"

		return goTypeRecursive(prefix, t.Elem())
	case reflect.Array:
		prefix += fmt.Sprintf("[%d]", t.Len())

		return goTypeRecursive(prefix, t.Elem())
	case reflect.Pointer:
		prefix += "*"

		return goTypeRecursive(prefix, t.Elem())
	case reflect.Interface:
		in := t.Name()
		if in == "" {
			in = "any"
		}

		return fmt.Sprintf("%s%s", prefix, in)
	case reflect.Struct:
		pkgpath := t.PkgPath()
		if pkgpath != "" {
			pkg, err := build.Import(pkgpath, ".", build.FindOnly)
			if err == nil && pkg.Goroot {
				return fmt.Sprintf("%s%s.%s", prefix, path.Base(t.PkgPath()), t.Name())
			}
		}

		return fmt.Sprintf("%s%s", prefix, t.Name())
	case reflect.Map:
		return fmt.Sprintf("map[%s]%s", goTypeRecursive("", t.Key()), goTypeRecursive("", t.Elem()))
	default:
		return fmt.Sprintf("%s%s", prefix, t.Kind().String())
	}
}

func tsType(t reflect.Type) string {
	return tsTypeRecursive("", t, "")
}

func tsTypeRecursive(prefix string, t reflect.Type, postfix string) string {
	// we need a hacky fix to handle correctly json.RawMessage and kit.RawMessage in auto-generated code
	// of the stubs
	switch t.String() {
	case "time.Time":
		return fmt.Sprintf("%sstring", prefix)
	case "json.RawMessage":
		return fmt.Sprintf("%s%s", prefix, "any")
	case "kit.RawMessage":
		return fmt.Sprintf("%s%s", prefix, "any")
	}

	//nolint:exhaustive
	switch t.Kind() {
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return fmt.Sprintf("%s%s%s", prefix, "string", postfix)
		}
		postfix += "[]"

		return tsTypeRecursive(prefix, t.Elem(), postfix)
	case reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return fmt.Sprintf("%s%s%s", prefix, "string", postfix)
		}
		postfix += fmt.Sprintf("[%d]", t.Len())

		return tsTypeRecursive(prefix, t.Elem(), postfix)
	case reflect.Pointer:
		return tsTypeRecursive(prefix, t.Elem(), postfix)
	case reflect.Struct:
		return fmt.Sprintf("%s%s%s", prefix, t.Name(), postfix)
	case reflect.Map:
		return fmt.Sprintf("{[key: %s]: %s}",
			tsTypeRecursive("", t.Key(), ""),
			tsTypeRecursive("", t.Elem(), ""),
		)
	case reflect.Interface:
		return fmt.Sprintf("%s%s%s", prefix, "any", postfix)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%s%s%s", prefix, "number", postfix)
	case reflect.Bool:
		return fmt.Sprintf("%s%s%s", prefix, "boolean", postfix)
	default:
		return fmt.Sprintf("%s%s%s", prefix, t.Kind().String(), postfix)
	}
}

func strAppend(arr []string, elem string) []string {
	return append(arr, elem)
}

func strEmptySlice() []string {
	return []string{}
}

var pathParamRegEX = regexp.MustCompile(`{([^}]+)}`)

func tsReplacePathParams(path string, prefix string) string {
	return pathParamRegEX.ReplaceAllStringFunc(path, func(s string) string {
		return fmt.Sprintf(`${%s%s}`, prefix, strings.Trim(s, "{}"))
	})
}
