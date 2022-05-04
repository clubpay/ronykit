package reflector

import (
	"fmt"
	"reflect"
	"unsafe"

	"github.com/clubpay/ronykit"
)

var (
	registered          = map[reflect.Type]*reflected{}
	errInvalidFieldType = func(s string) error { return fmt.Errorf("the field type does not match: %s", s) }
)

// emptyInterface is the header for an interface{} value.
type emptyInterface struct {
	typ  uint64
	word unsafe.Pointer
}

type fieldInfo struct {
	name   string
	offset uintptr
	typ    reflect.Type
	unsafe bool
}

func (f fieldInfo) Kind() reflect.Kind {
	if f.typ == nil {
		return reflect.Invalid
	}

	return f.typ.Kind()
}

type reflected struct {
	obj map[string]fieldInfo
	enc ronykit.Encoding
}

type Object struct {
	m    ronykit.Message
	data map[string]fieldInfo
	enc  ronykit.Encoding
}

func (o Object) Encoding() ronykit.Encoding {
	return o.enc
}

func (o Object) Message() ronykit.Message {
	return o.m
}

func (o Object) GetInt(fieldName string) (int, error) {
	fi := o.data[fieldName]
	if k := fi.Kind(); k != reflect.Int {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&o.m)).word, fi.offset)

	return *(*int)(ptr), nil
}

func (o Object) GetIntDefault(fieldName string, def int) int {
	v, err := o.GetInt(fieldName)
	if err != nil {
		return def
	}

	return v
}

func (o Object) GetUInt(fieldName string) (uint, error) {
	fi := o.data[fieldName]
	if k := fi.Kind(); k != reflect.Uint {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&o.m)).word, fi.offset)

	return *(*uint)(ptr), nil
}

func (o Object) GetUIntDefault(fieldName string, def uint) uint {
	v, err := o.GetUInt(fieldName)
	if err != nil {
		return def
	}

	return v
}

func (o Object) GetInt64(fieldName string) (int64, error) {
	fi := o.data[fieldName]
	if k := fi.Kind(); k != reflect.Int64 {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&o.m)).word, fi.offset)

	return *(*int64)(ptr), nil
}

func (o Object) GetInt64Default(fieldName string, def int64) int64 {
	v, err := o.GetInt64(fieldName)
	if err != nil {
		return def
	}

	return v
}

func (o Object) GetUInt64(fieldName string) (uint64, error) {
	fi := o.data[fieldName]
	if k := fi.Kind(); k != reflect.Uint64 {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&o.m)).word, fi.offset)

	return *(*uint64)(ptr), nil
}

func (o Object) GetUInt64Default(fieldName string, def uint64) uint64 {
	v, err := o.GetUInt64(fieldName)
	if err != nil {
		return def
	}

	return v
}

func (o Object) GetInt32(fieldName string) (int32, error) {
	fi := o.data[fieldName]
	if k := fi.Kind(); k != reflect.Int32 {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&o.m)).word, fi.offset)

	return *(*int32)(ptr), nil
}

func (o Object) GetInt32Default(fieldName string, def int32) int32 {
	v, err := o.GetInt32(fieldName)
	if err != nil {
		return def
	}

	return v
}

func (o Object) GetUInt32(fieldName string) (uint32, error) {
	fi := o.data[fieldName]
	if k := fi.Kind(); k != reflect.Uint32 {
		return 0, errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&o.m)).word, fi.offset)

	return *(*uint32)(ptr), nil
}

func (o Object) GetUInt32Default(fieldName string, def uint32) uint32 {
	v, err := o.GetUInt32(fieldName)
	if err != nil {
		return def
	}

	return v
}

func (o Object) GetString(fieldName string) (string, error) {
	fi := o.data[fieldName]
	if k := fi.Kind(); k != reflect.String {
		return "", errInvalidFieldType(k.String())
	}

	ptr := unsafe.Add((*emptyInterface)(unsafe.Pointer(&o.m)).word, fi.offset)

	return *(*string)(ptr), nil
}

func (o Object) GetStringDefault(fieldName string, def string) string {
	v, err := o.GetString(fieldName)
	if err != nil {
		return def
	}

	return v
}
