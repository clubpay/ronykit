package reflector

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/clubpay/ronykit/kit"
)

var (
	registered          = map[reflect.Type]*Reflected{}
	errInvalidFieldType = func(s string) error { return fmt.Errorf("the field type does not match: %s", s) }
)

// emptyInterface is the header for an any value.
type emptyInterface struct {
	_    uint64
	word unsafe.Pointer
}

type FieldInfo struct {
	idx    int
	f      reflect.StructField
	name   string
	offset uintptr
	typ    reflect.Type
	unsafe bool
}

func (f FieldInfo) Kind() reflect.Kind {
	if f.typ == nil {
		return reflect.Invalid
	}

	return f.typ.Kind()
}

func (f FieldInfo) Type() reflect.StructField {
	return f.f
}

type Fields map[string]FieldInfo

func (fields Fields) Get(m kit.Message, fieldName string) any {
	fi := fields[fieldName]
	mVal := reflect.Indirect(reflect.ValueOf(m)).Field(fi.idx)
	switch fi.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return mVal.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return mVal.Uint()
	case reflect.String:
		return mVal.String()
	case reflect.Float64, reflect.Float32:
		return mVal.Float()
	case reflect.Bool:
		return mVal.Bool()
	}

	if !mVal.CanInterface() {
		return nil
	}

	return mVal
}

func (fields Fields) GetInt(m kit.Message, fieldName string) (int, error) {
	fi := fields[fieldName]
	if k := fi.Kind(); k != reflect.Int {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&m)).word, fi.offset)

	return *(*int)(ptr), nil
}

func (fields Fields) GetIntDefault(m kit.Message, fieldName string, def int) int {
	v, err := fields.GetInt(m, fieldName)
	if err != nil {
		return def
	}

	return v
}

func (fields Fields) GetUInt(m kit.Message, fieldName string) (uint, error) {
	fi := fields[fieldName]
	if k := fi.Kind(); k != reflect.Uint {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&m)).word, fi.offset)

	return *(*uint)(ptr), nil
}

func (fields Fields) GetUIntDefault(m kit.Message, fieldName string, def uint) uint {
	v, err := fields.GetUInt(m, fieldName)
	if err != nil {
		return def
	}

	return v
}

func (fields Fields) GetInt64(m kit.Message, fieldName string) (int64, error) {
	fi := fields[fieldName]
	if k := fi.Kind(); k != reflect.Int64 {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&m)).word, fi.offset)

	return *(*int64)(ptr), nil
}

func (fields Fields) GetInt64Default(m kit.Message, fieldName string, def int64) int64 {
	v, err := fields.GetInt64(m, fieldName)
	if err != nil {
		return def
	}

	return v
}

func (fields Fields) GetUInt64(m kit.Message, fieldName string) (uint64, error) {
	fi := fields[fieldName]
	if k := fi.Kind(); k != reflect.Uint64 {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&m)).word, fi.offset)

	return *(*uint64)(ptr), nil
}

func (fields Fields) GetUInt64Default(m kit.Message, fieldName string, def uint64) uint64 {
	v, err := fields.GetUInt64(m, fieldName)
	if err != nil {
		return def
	}

	return v
}

func (fields Fields) GetInt32(m kit.Message, fieldName string) (int32, error) {
	fi := fields[fieldName]
	if k := fi.Kind(); k != reflect.Int32 {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&m)).word, fi.offset)

	return *(*int32)(ptr), nil
}

func (fields Fields) GetInt32Default(m kit.Message, fieldName string, def int32) int32 {
	v, err := fields.GetInt32(m, fieldName)
	if err != nil {
		return def
	}

	return v
}

func (fields Fields) GetUInt32(m kit.Message, fieldName string) (uint32, error) {
	fi := fields[fieldName]
	if k := fi.Kind(); k != reflect.Uint32 {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&m)).word, fi.offset)

	return *(*uint32)(ptr), nil
}

func (fields Fields) GetUInt32Default(m kit.Message, fieldName string, def uint32) uint32 {
	v, err := fields.GetUInt32(m, fieldName)
	if err != nil {
		return def
	}

	return v
}

func (fields Fields) GetString(m kit.Message, fieldName string) (string, error) {
	fi := fields[fieldName]
	if k := fi.Kind(); k != reflect.String {
		return "", errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&m)).word, fi.offset)

	return *(*string)(ptr), nil
}

func (fields Fields) GetStringDefault(m kit.Message, fieldName string, def string) string {
	v, err := fields.GetString(m, fieldName)
	if err != nil {
		return def
	}

	return v
}

func (fields Fields) WalkFields(cb func(key string, f FieldInfo)) {
	for k, f := range fields {
		cb(k, f)
	}
}

type Reflected struct {
	obj   Fields
	byTag map[string]Fields
	typ   reflect.Type
}

func (r Reflected) ByTag(t string) (Fields, bool) {
	f, ok := r.byTag[t]

	return f, ok
}

func (r Reflected) Obj() Fields {
	return r.obj
}

func (r Reflected) Type() reflect.Type {
	return r.typ
}
