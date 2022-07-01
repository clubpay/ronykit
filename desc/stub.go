package desc

import (
	"fmt"
	"reflect"
)

type DTO struct {
	Comments []string
	Name     string
	Type     string
	Fields   []DTOField
}

type DTOField struct {
	Name string
	Type string
	Tags []DTOFieldTag
}

type DTOFieldTag struct {
	Name  string
	Value string
}

type RESTMethod struct {
	Name     string
	Method   string
	Path     string
	Request  DTO
	Response []DTO
}

type RPCMethod struct {
	Name      string
	Predicate string
	Request   DTO
	Response  []DTO
}

type Stub struct {
	tags  []string
	DTOs  map[string]DTO
	RESTs []RESTMethod
	RPCs  []RPCMethod
}

func NewStub(tags ...string) Stub {
	return Stub{
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
				Name: ft.Name,
				Type: typ("", ft.Type),
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
	dto, ok := d.DTOs[mTyp.Name()]

	return dto, ok
}
