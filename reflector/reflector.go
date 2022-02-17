package reflector

import (
	"fmt"
	"reflect"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/utils"
)

type Reflector struct {
	tagName string

	cacheMtx utils.SpinLock
	cache    map[string]map[string]fieldInfo
}

func New() *Reflector {
	return &Reflector{
		cache: map[string]map[string]fieldInfo{},
	}
}

func (r *Reflector) extractInfo(m ronykit.Message) map[string]fieldInfo {
	rVal := reflect.ValueOf(m)
	rType := rVal.Type()
	if rType.Kind() != reflect.Ptr {
		panic("x must be a pointer to struct")
	}
	if rVal.Elem().Kind() != reflect.Struct {
		panic("x must be a pointer to struct")
	}

	cachedObj := map[string]fieldInfo{}
	for i := 0; i < reflect.Indirect(rVal).NumField(); i++ {
		f := reflect.Indirect(rVal).Type().Field(i)
		if f.IsExported() {
			cachedObj[f.Name] = fieldInfo{
				offset: f.Offset,
				typ:    f.Type,
			}
		}
	}

	r.cacheMtx.Lock()
	r.cache[rType.Name()] = cachedObj
	r.cacheMtx.Unlock()

	return cachedObj
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

var (
	ErrNotExported = fmt.Errorf("not exported")
	ErrNoField     = fmt.Errorf("field not exists")
)
