package silverhttp

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/std/gateways/silverhttp/httpmux"
	"github.com/goccy/go-reflect"
)

type (
	Params      = httpmux.Params
	DecoderFunc = func(bag Params, data []byte) (kit.Message, error)
)

// emptyInterface is the header for an interface{} value.
type emptyInterface struct {
	_    uint64
	word unsafe.Pointer
}

type paramCaster struct {
	offset uintptr
	name   string
	opt    string
	typ    reflect.Type
}

func reflectDecoder(enc kit.Encoding, factory kit.MessageFactoryFunc) DecoderFunc {
	switch factory().(type) {
	case kit.RawMessage:
		return func(bag Params, data []byte) (kit.Message, error) {
			v := kit.RawMessage{}
			v.CopyFrom(data)

			return v, nil
		}
	default:
	}

	tagKey := enc.Tag()
	if tagKey == "" {
		tagKey = kit.JSON.Tag()
	}

	rVal := reflect.ValueOf(factory())
	if rVal.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("%s must be a pointer to struct", rVal.String()))
	}
	rVal = rVal.Elem()
	if rVal.Kind() != reflect.Struct {
		panic(fmt.Sprintf("%s must be a pointer to struct", rVal.String()))
	}

	pcs := extractFields(rVal, tagKey)

	return func(bag Params, data []byte) (kit.Message, error) {
		var (
			v   = factory()
			err error
		)

		if len(data) > 0 {
			err = kit.UnmarshalMessage(data, v)
			if err != nil {
				return nil, err
			}
		}

		for idx := range pcs {
			x := bag.ByName(pcs[idx].name)
			if x == "" {
				continue
			}

			ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&v)).word, pcs[idx].offset)

			switch pcs[idx].typ.Kind() {
			default:
				// simply ignore
			case reflect.Int64:
				*(*int64)(ptr) = utils.StrToInt64(x)
			case reflect.Int32:
				*(*int32)(ptr) = utils.StrToInt32(x)
			case reflect.Uint64:
				*(*uint64)(ptr) = utils.StrToUInt64(x)
			case reflect.Uint32:
				*(*uint32)(ptr) = utils.StrToUInt32(x)
			case reflect.Float64:
				*(*float64)(ptr) = utils.StrToFloat64(x)
			case reflect.Float32:
				*(*float32)(ptr) = utils.StrToFloat32(x)
			case reflect.Int:
				*(*int)(ptr) = utils.StrToInt(x)
			case reflect.Uint:
				*(*uint)(ptr) = utils.StrToUInt(x)
			case reflect.Slice:
				switch pcs[idx].typ.Elem().Kind() {
				default:
					// simply ignore
				case reflect.Uint8:
					*(*[]byte)(ptr) = utils.S2B(x)
				}
			case reflect.String:
				*(*string)(ptr) = string(utils.S2B(x))
			case reflect.Bool:
				if strings.ToLower(x) == "true" {
					*(*bool)(ptr) = true
				}
			}
		}

		return v.(kit.Message), nil //nolint:forcetypeassert
	}
}

func extractFields(rVal reflect.Value, tagKey string) []paramCaster {
	var pcs []paramCaster
	for i := 0; i < rVal.NumField(); i++ {
		f := rVal.Type().Field(i)
		if f.Type.Kind() == reflect.Struct && f.Anonymous {
			pcs = append(pcs, extractFields(rVal.Field(i), tagKey)...)
		} else {
			if tagValue := f.Tag.Get(tagKey); tagValue != "" {
				valueParts := strings.Split(tagValue, ",")
				if len(valueParts) == 1 {
					valueParts = append(valueParts, "")
				}

				pcs = append(
					pcs,
					paramCaster{
						offset: f.Offset,
						name:   valueParts[0],
						opt:    valueParts[1],
						typ:    f.Type,
					},
				)
			}
		}
	}

	return pcs
}
