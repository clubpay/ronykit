package fasthttp

import (
	"encoding/json"
	"reflect"
	"strings"
	"unsafe"

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
	Selector    struct {
		Method        string
		Path          string
		Predicate     string
		CustomDecoder DecoderFunc
	}
)

func (sd Selector) Generate(f ronykit.MessageFactory) ronykit.RouteSelector {
	route := &routeSelector{
		method:    sd.Method,
		path:      sd.Path,
		predicate: sd.Predicate,
	}
	if sd.CustomDecoder != nil {
		route.decoder = sd.CustomDecoder
	} else {
		route.decoder = reflectDecoder(f)
	}

	return route
}

var (
	_ ronykit.RouteSelector     = routeSelector{}
	_ ronykit.RESTRouteSelector = routeSelector{}
	_ ronykit.RPCRouteSelector  = routeSelector{}
)

// routeSelector selector implements ronykit.RouteSelector and
// also ronykit.RPCRouteSelector and ronykit.RESTRouteSelector
type routeSelector struct {
	method    string
	path      string
	predicate string
	decoder   DecoderFunc
}

func (r routeSelector) GetMethod() string {
	return r.method
}

func (r routeSelector) GetPath() string {
	return r.path
}

func (r routeSelector) GetPredicate() string {
	return r.predicate
}

func (r routeSelector) Query(q string) interface{} {
	switch q {
	case queryDecoder:
		return r.decoder
	case queryMethod:
		return r.method
	case queryPath:
		return r.path
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

func reflectDecoder(factory ronykit.MessageFactory) DecoderFunc {
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
