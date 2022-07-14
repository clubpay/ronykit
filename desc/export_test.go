package desc

import "reflect"

var (
	TypeOf  = typ
	NewStub = newStub
)

func (d *Stub) AddDTO(mTyp reflect.Type, isErr bool) error {
	return d.addDTO(mTyp, isErr)
}
