package reflector

import "reflect"

type fieldInfo struct {
	offset uintptr
	typ    reflect.Type
}
