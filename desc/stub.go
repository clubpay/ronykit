package desc

import (
	"fmt"
	"reflect"

	"github.com/clubpay/ronykit"
)

// DTO represents the description of Data Object Transfer of the Stub
type DTO struct {
	Comments []string
	Name     string
	Type     string
	Fields   []DTOField
}

// DTOField represents description of a field of the DTO
type DTOField struct {
	Name     string
	Type     string
	Embedded bool
	IsDTO    bool
	Tags     []DTOFieldTag
}

// DTOFieldTag represents description of a tag of the DTOField
type DTOFieldTag struct {
	Name  string
	Value string
}

// RESTMethod represents description of a Contract with ronykit.RESTRouteSelector.
type RESTMethod struct {
	Name     string
	Method   string
	Path     string
	Request  DTO
	Response []DTO
}

// RPCMethod represents description of a Contract with ronykit.RPCRouteSelector
type RPCMethod struct {
	Name      string
	Predicate string
	Request   DTO
	Response  []DTO
	ronykit.IncomingRPCContainer
	ronykit.OutgoingRPCContainer
}

// Stub represents description of a stub of the service described by Service descriptor.
type Stub struct {
	tags  []string
	Pkg   string
	Name  string
	DTOs  map[string]DTO
	RESTs []RESTMethod
	RPCs  []RPCMethod
}

func newStub(tags ...string) *Stub {
	return &Stub{
		tags: tags,
		DTOs: map[string]DTO{},
	}
}

func (d *Stub) addDTO(mTyp reflect.Type) error {
	dto := DTO{}
	if mTyp.Kind() == reflect.Ptr {
		mTyp = mTyp.Elem()
	}

	dto.Name = mTyp.Name()
	switch mTyp.Kind() {
	case reflect.Struct:
		for i := 0; i < mTyp.NumField(); i++ {
			ft := mTyp.Field(i)
			k := ft.Type.Kind()

			switch {
			case k == reflect.Struct:
				err := d.addDTO(ft.Type)
				if err != nil {
					return err
				}
			case k == reflect.Ptr && ft.Type.Elem().Kind() == reflect.Struct:
				err := d.addDTO(ft.Type.Elem())
				if err != nil {
					return err
				}
			case k == reflect.Interface:
				// we ignore interface types in DTOs
				// FIXME: maybe we can implement some dummy struct which implements the interface ?
				continue
			}

			dto.Type = typ("", mTyp)
			dtoF := DTOField{
				Name:     ft.Name,
				Type:     typ("", ft.Type),
				Embedded: ft.Anonymous,
				IsDTO:    k == reflect.Struct,
			}

			for _, t := range d.tags {
				v, ok := ft.Tag.Lookup(t)
				if ok {
					dtoF.Tags = append(dtoF.Tags,
						DTOFieldTag{
							Name:  t,
							Value: v,
						},
					)
				}
			}

			dto.Fields = append(dto.Fields, dtoF)
		}
	case reflect.Interface:
		// we ignore interface types in DTOs
		// FIXME: maybe we can implement some dummy struct which implements the interface ?

		return fmt.Errorf("we don't support interface types as DTO")
	default:
		dto.Type = typ("", mTyp)
	}

	d.DTOs[dto.Name] = dto

	return nil
}

func (d *Stub) getDTO(mTyp reflect.Type) (DTO, bool) {
	if mTyp.Kind() == reflect.Ptr {
		mTyp = mTyp.Elem()
	}

	dto, ok := d.DTOs[mTyp.Name()]

	return dto, ok
}
