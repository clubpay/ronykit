package fasthttp

import (
	"reflect"
	"strings"
	"unsafe"

	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/utils"
)

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the mux.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) string {
	for _, p := range ps {
		if p.Key == name {
			return p.Value
		}
	}

	return ""
}

type (
	DecoderFunc func(bag Params, data []byte) ronykit.Message
)

var (
	_ ronykit.RouteSelector     = Selector{}
	_ ronykit.RESTRouteSelector = Selector{}
	_ ronykit.RPCRouteSelector  = Selector{}
)

// Selector implements ronykit.RouteSelector and
// also ronykit.RPCRouteSelector and ronykit.RESTRouteSelector
type Selector struct {
	Method    string
	Path      string
	Predicate string
	Decoder   DecoderFunc
}

func (r Selector) GetMethod() string {
	return r.Method
}

func (r Selector) GetPath() string {
	return r.Path
}

func (r Selector) GetPredicate() string {
	return r.Predicate
}

func (r Selector) Query(q string) interface{} {
	switch q {
	case queryDecoder:
		return r.Decoder
	case queryMethod:
		return r.Method
	case queryPath:
		return r.Path
	}

	return nil
}

var tagKey = "paramName"

// SetTag set the tag name which ReflectDecoder looks to extract parameters from Path and Query params.
// Default value: paramName
func SetTag(tag string) {
	tagKey = tag
}

// emptyInterface is the header for an interface{} value.
type emptyInterface struct {
	typ  uint64
	word unsafe.Pointer
}

type paramCaster struct {
	offset uintptr
	name   string
	typ    reflect.Type
}

func reflectDecoder(factory ronykit.MessageFactoryFunc) DecoderFunc {
	rVal := reflect.ValueOf(factory())
	rType := rVal.Type()
	if rType.Kind() != reflect.Ptr {
		panic("x must be a pointer to struct")
	}
	if rVal.Elem().Kind() != reflect.Struct {
		panic("x must be a pointer to struct")
	}

	var pcs []paramCaster
	for i := 0; i < reflect.Indirect(rVal).NumField(); i++ {
		f := reflect.Indirect(rVal).Type().Field(i)
		if tagValue := f.Tag.Get(tagKey); tagValue != "" {
			pcs = append(
				pcs,
				paramCaster{
					offset: f.Offset,
					name:   strings.Split(tagValue, ",")[0],
					typ:    f.Type,
				},
			)
		}
	}

	return func(bag Params, data []byte) ronykit.Message {
		v := factory()

		if len(data) > 0 {
			_ = json.Unmarshal(data, v)
		}

		for idx := range pcs {
			x := bag.ByName(pcs[idx].name)
			if x == "" {
				continue
			}

			ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&v)).word, pcs[idx].offset)
			switch pcs[idx].typ.Kind() {
			case reflect.Int64:
				*(*int64)(ptr) = utils.StrToInt64(x)
			case reflect.Int32:
				*(*int32)(ptr) = utils.StrToInt32(x)
			case reflect.Uint64:
				*(*uint64)(ptr) = utils.StrToUInt64(x)
			case reflect.Uint32:
				*(*uint32)(ptr) = utils.StrToUInt32(x)
			case reflect.Int:
				*(*int)(ptr) = utils.StrToInt(x)
			case reflect.Uint:
				*(*uint)(ptr) = utils.StrToUInt(x)
			case reflect.String:
				// FixME: make this copy as an option
				*(*string)(ptr) = string(utils.S2B(x))
			}
		}

		return v.(ronykit.Message)
	}
}
