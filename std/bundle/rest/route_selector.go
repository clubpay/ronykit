package rest

import (
	"reflect"
	"unsafe"

	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest/mux"
	"github.com/ronaksoft/ronykit/utils"
	"github.com/valyala/fasthttp"
)

type routeSelector struct {
	method  string
	path    string
	decoder mux.DecoderFunc
}

func Route(method string, path string) *routeSelector {
	return &routeSelector{
		method: method,
		path:   path,
	}
}

func Get(path string) *routeSelector {
	return Route(fasthttp.MethodGet, path)
}

func Post(path string) *routeSelector {
	return Route(fasthttp.MethodPost, path)
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

func (r *routeSelector) WithFactory(f ronykit.MessageFactory) *routeSelector {
	r.decoder = ReflectDecoder(f)

	return r
}

func (r *routeSelector) WithDecoder(d mux.DecoderFunc) *routeSelector {
	r.decoder = d

	return r
}

var tagName = "paramName"

// SetTag set the tag name which ReflectDecoder looks to extract parameters from Path and Query params.
// Default value: paramName
func SetTag(tag string) {
	tagName = tag
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

func ReflectDecoder(factory ronykit.MessageFactory) mux.DecoderFunc {
	x := factory()
	rVal := reflect.ValueOf(x)
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
		if tagName := f.Tag.Get(tagName); tagName != "" {
			pcs = append(
				pcs,
				paramCaster{
					offset: f.Offset,
					name:   tagName,
					typ:    f.Type,
				},
			)
		}
	}

	return func(bag mux.Params, data []byte) ronykit.Message {
		v := factory()

		if len(data) > 0 {
			_ = json.Unmarshal(data, v)
		}

		for idx := range pcs {
			ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&v)).word, pcs[idx].offset)
			switch pcs[idx].typ.Kind() {
			case reflect.Int64:
				*(*int64)(ptr) = utils.StrToInt64(bag.ByName(pcs[idx].name))
			case reflect.Int32:
				*(*int32)(ptr) = int32(utils.StrToInt64(bag.ByName(pcs[idx].name)))
			case reflect.Uint64:
				*(*uint64)(ptr) = uint64(utils.StrToInt64(bag.ByName(pcs[idx].name)))
			case reflect.Uint32:
				*(*uint32)(ptr) = uint32(utils.StrToInt64(bag.ByName(pcs[idx].name)))
			case reflect.Int:
				*(*int)(ptr) = int(utils.StrToInt64(bag.ByName(pcs[idx].name)))
			case reflect.Uint:
				*(*uint)(ptr) = uint(utils.StrToInt64(bag.ByName(pcs[idx].name)))
			case reflect.String:
				*(*string)(ptr) = bag.ByName(pcs[idx].name)
			}
		}

		return v.(ronykit.Message)
	}
}
