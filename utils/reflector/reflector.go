package reflector

import (
	"encoding"
	"fmt"
	"reflect"
	"sync"

	"github.com/clubpay/ronykit"
)

var (
	ErrNotExported = fmt.Errorf("not exported")
	ErrNoField     = fmt.Errorf("field not exists")
)

type Reflector struct {
	tagName string

	cacheMtx sync.RWMutex
	cache    map[reflect.Type]*reflected
}

func New() *Reflector {
	return &Reflector{
		cache: map[reflect.Type]*reflected{},
	}
}

// Register registers the message then reflector will be much faster. You should call
// it concurrently.
func Register(m ronykit.Message) {
	if m == nil {
		return
	}

	mVal, err := getValue(m)
	if err != nil {
		return
	}
	mType := mVal.Type()
	registered[mType] = &reflected{
		obj: destruct(mVal),
		enc: getEncoding(m),
	}
}

func getEncoding(m ronykit.Message) ronykit.Encoding {
	var e ronykit.Encoding

	_, ok := m.(interface {
		Marshal() ([]byte, error)
		Unmarshal([]byte) error
	})
	if ok {
		e |= ronykit.CustomDefined
	}

	_, ok = m.(interface {
		MarshalJSON() ([]byte, error)
		UnmarshalJSON([]byte) error
	})
	if ok {
		e |= ronykit.JSON
	}

	_, ok = m.(interface {
		MarshalProto() ([]byte, error)
		UnmarshalProto([]byte) error
	})
	if ok {
		e |= ronykit.Proto
	}

	_, ok = m.(interface {
		encoding.BinaryMarshaler
		encoding.BinaryUnmarshaler
	})
	if ok {
		e |= ronykit.Binary
	}

	_, ok = m.(interface {
		encoding.TextMarshaler
		encoding.TextUnmarshaler
	})
	if ok {
		e |= ronykit.Text
	}

	return e
}

func getValue(m ronykit.Message) (reflect.Value, error) {
	mVal := reflect.Indirect(reflect.ValueOf(m))
	if mVal.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("message is not a struct")
	}

	return mVal, nil
}

func destruct(mVal reflect.Value) map[string]fieldInfo {
	mType := mVal.Type()
	data := map[string]fieldInfo{}

	for i := 0; i < mType.NumField(); i++ {
		ft := mType.Field(i)
		if !ft.IsExported() {
			continue
		}
		fi := fieldInfo{
			name:   ft.Name,
			offset: ft.Offset,
			typ:    ft.Type,
		}

		switch ft.Type.Kind() {
		case reflect.Map, reflect.Slice, reflect.Ptr, reflect.Interface,
			reflect.Array, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.UnsafePointer:
		default:
			fi.unsafe = true
		}
		data[ft.Name] = fi
	}

	return data
}

func (r *Reflector) Load(m ronykit.Message) (Object, error) {
	mVal, err := getValue(m)
	if err != nil {
		return Object{}, err
	}
	mType := mVal.Type()
	cachedData := registered[mType]
	if cachedData == nil {
		r.cacheMtx.RLock()
		cachedData = r.cache[mType]
		r.cacheMtx.RUnlock()
		if cachedData == nil {
			cachedData = &reflected{
				obj: destruct(mVal),
				enc: getEncoding(m),
			}
			r.cacheMtx.Lock()
			r.cache[mType] = cachedData
			r.cacheMtx.Unlock()
		}
	}

	return Object{
		m:    m,
		data: cachedData.obj,
		enc:  cachedData.enc,
	}, nil
}

func (r *Reflector) Get(m ronykit.Message, fieldName string) (interface{}, error) {
	e, err := getValue(m)
	if err != nil {
		return nil, err
	}

	f, ok := e.Type().FieldByName(fieldName)
	if !ok {
		return nil, ErrNoField
	}
	if !f.IsExported() {
		return nil, ErrNotExported
	}

	return e.FieldByName(fieldName).Interface(), nil
}

func (r *Reflector) GetString(m ronykit.Message, fieldName string) (string, error) {
	e, err := getValue(m)
	if err != nil {
		return "", err
	}
	f, ok := e.Type().FieldByName(fieldName)
	if !ok {
		return "", ErrNoField
	}
	if !f.IsExported() {
		return "", ErrNotExported
	}

	return e.FieldByName(fieldName).String(), nil
}

func (r *Reflector) GetInt(m ronykit.Message, fieldName string) (int64, error) {
	e, err := getValue(m)
	if err != nil {
		return 0, err
	}
	f, ok := e.Type().FieldByName(fieldName)
	if !ok {
		return 0, ErrNoField
	}
	if !f.IsExported() {
		return 0, ErrNotExported
	}

	return e.FieldByName(fieldName).Int(), nil
}
