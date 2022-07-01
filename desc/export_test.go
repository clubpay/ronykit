package desc

import "reflect"

var (
	TypeOf  = typ
	NewStub = newStub
)

func (d *Stub) AddDTO(mTyp reflect.Type) error {
	return d.addDTO(mTyp)
}
