package fasthttp

import (
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

type DecoderFunc func(bag Params, data []byte) (kit.Message, error)

func (b *bundle) genParams(ctx *fasthttp.RequestCtx) Params {
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
	ctx.QueryArgs().VisitAll(
		func(key, value []byte) {
			params = append(
				params,
				Param{
					Key:   utils.B2S(key),
					Value: utils.B2S(value),
				},
			)
		},
	)

	ctx.PostArgs().VisitAll(
		func(key, value []byte) {
			params = append(
				params,
				Param{
					Key:   utils.B2S(key),
					Value: utils.B2S(value),
				},
			)
		},
	)

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
		panic("x must be a pointer to struct")
	}
	rVal = rVal.Elem()
	if rVal.Kind() != reflect.Struct {
		panic("x must be a pointer to struct")
	}

	var pcs []paramCaster

	for i := 0; i < rVal.NumField(); i++ {
		f := rVal.Type().Field(i)
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
