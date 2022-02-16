package reflector

import (
	"reflect"
	"unsafe"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/utils"
)

// emptyInterface is the header for an interface{} value.
type emptyInterface struct {
	typ  uint64
	word unsafe.Pointer
}

type Reflector struct {
	tagName string

	cacheMtx utils.SpinLock
	cache    map[string]map[string]fieldInfo
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
	f, ok := reflect.TypeOf(m).FieldByName(fieldName)
	if !ok {
		return nil
	}
	if !f.IsExported() {
		return nil
	}

	return reflect.ValueOf(m).FieldByName(fieldName).Interface()
}

//func (r *Reflector) Get(m ronykit.Message, fieldName string) interface{} {
//	mt := reflect.TypeOf(m).Name()
//	r.cacheMtx.Lock()
//	obj := r.cache[mt]
//	r.cacheMtx.Unlock()
//	if obj == nil {
//		obj = r.extractInfo(m)
//	}
//
//	f := obj[fieldName]
//	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&m)).word, f.offset)
//
//	switch f.typ.Kind() {
//	case reflect.Int64:
//		*(*int64)(ptr)
//	case reflect.Int32:
//		*(*int32)(ptr) = utils.StrToInt32(x)
//	case reflect.Uint64:
//		*(*uint64)(ptr) = utils.StrToUInt64(x)
//	case reflect.Uint32:
//		*(*uint32)(ptr) = utils.StrToUInt32(x)
//	case reflect.Int:
//		return *(*int)(ptr)
//	case reflect.Uint:
//		return *(*uint)(ptr)
//	case reflect.String:
//		return *(*string)(ptr)
//
//	}
//
//}
