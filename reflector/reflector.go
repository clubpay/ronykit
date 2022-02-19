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

var registered = map[reflect.Type]map[string]fieldInfo{}

type Reflector struct {
	tagName string

	cacheMtx sync.RWMutex
	cache    map[reflect.Type]map[string]fieldInfo
}

func New() *Reflector {
	return &Reflector{
		cache: map[reflect.Type]map[string]fieldInfo{},
	}
}

// Register registers the message then reflector will be much faster. You should call
// it concurrently.
func Register(m ronykit.Message) {
	if m == nil {
		return
	}

	mVal := getValue(m)
	mType := mVal.Type()
	registered[mType] = destruct(mVal)
}

func getValue(m ronykit.Message) reflect.Value {
	if reflect.TypeOf(m).Kind() != reflect.Ptr {
		panic("must be a pointer to struct")
	}
	mVal := reflect.Indirect(reflect.ValueOf(m))
	if mVal.Kind() != reflect.Struct {
		panic("must be a pointer to struct")
	}

	return mVal
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

func (r *Reflector) Load(m ronykit.Message) Object {
	mVal := getValue(m)
	mType := mVal.Type()
	cachedData := registered[mType]
	if cachedData == nil {
		r.cacheMtx.RLock()
		cachedData = r.cache[mType]
		r.cacheMtx.RUnlock()
		if cachedData == nil {
			cachedData = destruct(mVal)
			r.cacheMtx.Lock()
			r.cache[mType] = cachedData
			r.cacheMtx.Unlock()
		}
	}

	return Object{
		m:    m,
		data: cachedData,
	}
}

func (r *Reflector) Get(m ronykit.Message, fieldName string) interface{} {
	rVal := reflect.ValueOf(m)
	rType := rVal.Type()
	if rType.Kind() != reflect.Ptr {
		panic("x must be a pointer to struct")
	}
	e := rVal.Elem()
	if e.Kind() != reflect.Struct {
		panic("x must be a pointer to struct")
	}

	f, ok := e.Type().FieldByName(fieldName)
	if !ok {
		return nil
	}
	if !f.IsExported() {
		return nil
	}

	return e.FieldByName(fieldName).Interface()
}

func (r *Reflector) GetString(m ronykit.Message, fieldName string) (string, error) {
	rVal := reflect.ValueOf(m)
	rType := rVal.Type()
	if rType.Kind() != reflect.Ptr {
		panic("x must be a pointer to struct")
	}
	e := rVal.Elem()
	if e.Kind() != reflect.Struct {
		panic("x must be a pointer to struct")
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
	rVal := reflect.ValueOf(m)
	rType := rVal.Type()
	if rType.Kind() != reflect.Ptr {
		panic("x must be a pointer to struct")
	}
	e := rVal.Elem()
	if e.Kind() != reflect.Struct {
		panic("x must be a pointer to struct")
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
