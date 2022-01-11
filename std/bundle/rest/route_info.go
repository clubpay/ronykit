package rest

import (
	"reflect"
	"unsafe"

	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest/mux"
	"github.com/ronaksoft/ronykit/utils"
)

type routeData struct {
	method  string
	path    string
	decoder mux.DecoderFunc
}

func NewRouteData(method string, path string, decoder mux.DecoderFunc) *routeData {
	return &routeData{
		method:  method,
		path:    path,
		decoder: decoder,
	}
}

func (r routeData) Query(q string) interface{} {
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

func ReflectDecoder(factory func() interface{}) mux.DecoderFunc {
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
		if tagName := f.Tag.Get("paramName"); tagName != "" {
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
		_ = json.Unmarshal(data, v)

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
