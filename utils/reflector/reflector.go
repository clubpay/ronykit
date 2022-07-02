package reflector

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/clubpay/ronykit"
)

var (
	ErrNotExported = fmt.Errorf("not exported")
	ErrNoField     = fmt.Errorf("field not exists")
)

var noObject = Object{}

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
	}
}

func getValue(m ronykit.Message) (reflect.Value, error) {
	mVal := reflect.Indirect(reflect.ValueOf(m))
	if mVal.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("message is not a struct")
	}

	return mVal, nil
}

func destruct(mVal reflect.Value) map[string]FieldInfo {
	mType := mVal.Type()
	data := map[string]FieldInfo{}

	for i := 0; i < mType.NumField(); i++ {
		ft := mType.Field(i)
		if !ft.IsExported() {
			continue
		}
		fi := FieldInfo{
			f:      ft,
			name:   ft.Name,
			offset: ft.Offset,
			typ:    ft.Type,
		}

		switch ft.Type.Kind() {
		case reflect.Map, reflect.Slice, reflect.Ptr,
			reflect.Interface, reflect.Array, reflect.Chan,
			reflect.Complex64, reflect.Complex128, reflect.UnsafePointer:
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
		return noObject, err
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
			}
			r.cacheMtx.Lock()
			r.cache[mType] = cachedData
			r.cacheMtx.Unlock()
		}
	}

	return Object{
		m:      m,
		t:      mType,
		fields: cachedData.obj,
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
