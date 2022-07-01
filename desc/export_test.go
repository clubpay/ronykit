package desc

import "reflect"

var TypeOf = typ

func (d *Stub) AddDTO(mTyp reflect.Type) error {
	return d.addDTO(mTyp)
}
