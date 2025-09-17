package fasthttp

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/goccy/go-reflect"
	"github.com/valyala/fasthttp"
)

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the httpMux.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) string {
	for _, p := range ps {
		if p.Key == name {
			return string(utils.S2B(p.Value))
		}
	}

	return ""
}

type RequestCtx = fasthttp.RequestCtx

type DecoderFunc func(reqCtx *RequestCtx, data []byte) (kit.Message, error)

func GetParams(ctx *RequestCtx) Params {
	var params Params

	ctx.VisitUserValues(
		func(key []byte, value any) {
			switch v := value.(type) {
			default:
			case []byte:
				params = append(
					params,
					Param{
						Key:   utils.B2S(key),
						Value: utils.B2S(v),
					},
				)
			case string:
				params = append(
					params,
					Param{
						Key:   utils.B2S(key),
						Value: v,
					},
				)
			}
		},
	)

	// Walk over all the query params
	for key, value := range ctx.QueryArgs().All() {
		params = append(
			params,
			Param{
				Key:   strings.TrimSuffix(utils.B2S(key), "[]"),
				Value: utils.B2S(value),
			},
		)
	}

	for key, value := range ctx.PostArgs().All() {
		params = append(
			params,
			Param{
				Key:   utils.B2S(key),
				Value: utils.B2S(value),
			},
		)
	}

	return params
}

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
	case kit.MultipartFormMessage:
		return func(reqCtx *RequestCtx, data []byte) (kit.Message, error) {
			v := kit.MultipartFormMessage{}

			frm, err := reqCtx.Request.MultipartForm()
			if err != nil {
				return nil, err
			}

			v.SetForm(frm)

			return v, nil
		}
	case kit.RawMessage:
		return func(_ *RequestCtx, data []byte) (kit.Message, error) {
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

	return genDecoderFunc(factory, pcs...)
}

//nolint:cyclop,gocognit,gocyclo
func genDecoderFunc(factory kit.MessageFactoryFunc, pcs ...paramCaster) DecoderFunc {
	pcsMap := make(map[string]paramCaster, len(pcs))
	for _, pc := range pcs {
		pcsMap[pc.name] = pc
	}

	return func(reqCtx *RequestCtx, data []byte) (kit.Message, error) {
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

		bag := GetParams(reqCtx)
		// protect against too many query params
		if len(bag) > 64 {
			bag = bag[:64]
		}

		for idx := range bag {
			pc, ok := pcsMap[bag[idx].Key]
			if !ok {
				continue
			}

			x := bag[idx].Value
			if x == "" {
				continue
			}

			ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&v)).word, pc.offset)

			switch pc.typ.Kind() {
			default:
			// simply ignore
			case reflect.Ptr:
				switch pc.typ.Elem().Kind() {
				default:
				// simply ignore
				case reflect.Bool:
					if strings.ToLower(x) == "true" {
						*(**bool)(ptr) = utils.ValPtr(true)
					}
				case reflect.String:
					*(**string)(ptr) = utils.ValPtr(x)
				case reflect.Int64:
					*(**int64)(ptr) = utils.ValPtr(utils.StrToInt64(x))
				case reflect.Int32:
					*(**int32)(ptr) = utils.ValPtr(utils.StrToInt32(x))
				case reflect.Uint64:
					*(**uint64)(ptr) = utils.ValPtr(utils.StrToUInt64(x))
				case reflect.Uint32:
					*(**uint32)(ptr) = utils.ValPtr(utils.StrToUInt32(x))
				case reflect.Float64:
					*(**float64)(ptr) = utils.ValPtr(utils.StrToFloat64(x))
				case reflect.Float32:
					*(**float32)(ptr) = utils.ValPtr(utils.StrToFloat32(x))
				case reflect.Int:
					*(**int)(ptr) = utils.ValPtr(utils.StrToInt(x))
				case reflect.Uint:
					*(**uint)(ptr) = utils.ValPtr(utils.StrToUInt(x))
				}
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
				switch pc.typ.Elem().Kind() {
				default:
					// simply ignore
				case reflect.Int64:
					*(*[]int64)(ptr) = append(*(*[]int64)(ptr), utils.StrToInt64(x))
				case reflect.Int32:
					*(*[]int32)(ptr) = append(*(*[]int32)(ptr), utils.StrToInt32(x))
				case reflect.Uint64:
					*(*[]uint64)(ptr) = append(*(*[]uint64)(ptr), utils.StrToUInt64(x))
				case reflect.Uint32:
					*(*[]uint32)(ptr) = append(*(*[]uint32)(ptr), utils.StrToUInt32(x))
				case reflect.Float64:
					*(*[]float64)(ptr) = append(*(*[]float64)(ptr), utils.StrToFloat64(x))
				case reflect.Float32:
					*(*[]float32)(ptr) = append(*(*[]float32)(ptr), utils.StrToFloat32(x))
				case reflect.String:
					*(*[]string)(ptr) = append(*(*[]string)(ptr), x)
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
