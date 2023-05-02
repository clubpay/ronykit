package reflector

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"

	"github.com/clubpay/ronykit/kit"
)

var (
	ErrNotExported = fmt.Errorf("not exported")
	ErrNoField     = fmt.Errorf("field not exists")
)

type Reflector struct {
	cacheMtx sync.RWMutex
	cache    map[reflect.Type]*Reflected
}

func New() *Reflector {
	return &Reflector{
		cache: map[reflect.Type]*Reflected{},
	}
}

// Register registers the message then reflector will be much faster. You should call
// it concurrently.
func Register(m kit.Message, tags ...string) {
	if m == nil {
		return
	}

	mVal, err := getValue(m)
	if err != nil {
		return
	}
	mType := mVal.Type()
	registered[mType] = destruct(mType, tags...)
}

func getValue(m kit.Message) (reflect.Value, error) {
	mVal := reflect.Indirect(reflect.ValueOf(m))
	if mVal.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("message is not a struct")
	}

	return mVal, nil
}

func destruct(mType reflect.Type, tags ...string) *Reflected {
	r := &Reflected{
		obj:   Fields{},
		byTag: map[string]Fields{},
		typ:   mType,
	}

	for i := 0; i < mType.NumField(); i++ {
		ft := mType.Field(i)
		if ft.PkgPath != "" {
			continue
		}
		fi := FieldInfo{
			idx:    i,
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

		r.obj[fi.name] = fi
		for _, t := range tags {
			v, ok := ft.Tag.Lookup(t)
			if !ok {
				continue
			}
			idx := strings.IndexFunc(v, unicode.IsPunct)
			if idx != -1 {
				v = v[:idx]
			}
			if r.byTag[t] == nil {
				r.byTag[t] = Fields{}
			}
			r.byTag[t][v] = fi
		}
	}

	return r
}

func (r *Reflector) Load(m kit.Message, tags ...string) *Reflected {
	mType := reflect.Indirect(reflect.ValueOf(m)).Type()
	cachedData := registered[mType]
	if cachedData == nil {
		r.cacheMtx.RLock()
		cachedData = r.cache[mType]
		r.cacheMtx.RUnlock()
		if cachedData == nil {
			cachedData = destruct(mType, tags...)
			r.cacheMtx.Lock()
			r.cache[mType] = cachedData
			r.cacheMtx.Unlock()
		}
	}

	return cachedData
}

func (r *Reflector) Get(m kit.Message, fieldName string) (any, error) {
	e, err := getValue(m)
	if err != nil {
		return nil, err
	}

	f, ok := e.Type().FieldByName(fieldName)
	if !ok {
		return nil, ErrNoField
	}
	if f.PkgPath != "" {
		return nil, ErrNotExported
	}

	return e.FieldByName(fieldName).Interface(), nil
}

func (r *Reflector) GetString(m kit.Message, fieldName string) (string, error) {
	e, err := getValue(m)
	if err != nil {
		return "", err
	}
	f, ok := e.Type().FieldByName(fieldName)
	if !ok {
		return "", ErrNoField
	}
	if f.PkgPath != "" {
		return "", ErrNotExported
	}

	return e.FieldByName(fieldName).String(), nil
}

func (r *Reflector) GetInt(m kit.Message, fieldName string) (int64, error) {
	e, err := getValue(m)
	if err != nil {
		return 0, err
	}
	f, ok := e.Type().FieldByName(fieldName)
	if !ok {
		return 0, ErrNoField
	}
	if f.PkgPath != "" {
		return 0, ErrNotExported
	}

	return e.FieldByName(fieldName).Int(), nil
}
