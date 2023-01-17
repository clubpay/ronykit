package desc

import (
	"reflect"
	"strings"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/errors"
)

// DTO represents the description of Data Transfer Object of the Stub
type DTO struct {
	// Comments could be used by generators to print some useful information about this DTO
	Comments []string
	// Name is the name of this DTO struct
	Name   string
	Type   string
	IsErr  bool
	Fields []DTOField
}

func (dto DTO) CodeField() string {
	var fn string
	for _, f := range dto.Fields {
		x := strings.ToLower(f.Name)
		if f.Type != "int" {
			continue
		}
		if x == "code" {
			return f.Name
		}
		if strings.HasPrefix(f.Name, "code") {
			fn = f.Name
		}
	}

	return fn
}

func (dto DTO) ItemField() string {
	var fn string
	for _, f := range dto.Fields {
		x := strings.ToLower(f.Name)
		if f.Type != "string" {
			continue
		}
		if x == "item" || x == "items" {
			return f.Name
		}
		if strings.HasPrefix(f.Name, "item") {
			fn = f.Name
		}
	}

	return fn
}

// DTOField represents description of a field of the DTO
type DTOField struct {
	// Name of this field
	Name string
	// Type of this field and if this type is slice or map then it might have one or two
	// subtypes
	Type     string
	SubType1 string
	SubType2 string
	// If this field was an embedded field means fields are coming from an embedded DTO
	// If Embedded is TRUE then for sure IsDTO must be TRUE
	Embedded bool
	IsDTO    bool
	Tags     []DTOFieldTag
}

// DTOFieldTag represents description of a tag of the DTOField
type DTOFieldTag struct {
	Name  string
	Value string
}

// ErrorDTO represents description of a Data Object Transfer which is used to
// show an error case.
type ErrorDTO struct {
	Code int
	Item string
	DTO  DTO
}

// RESTMethod represents description of a Contract with kit.RESTRouteSelector.
type RESTMethod struct {
	Name           string
	Method         string
	Path           string
	Encoding       string
	Request        DTO
	Response       DTO
	PossibleErrors []ErrorDTO
}

func (rm *RESTMethod) addPossibleError(dto ErrorDTO) {
	for _, e := range rm.PossibleErrors {
		if e.Code == dto.Code {
			return
		}
	}
	rm.PossibleErrors = append(rm.PossibleErrors, dto)
}

// RPCMethod represents description of a Contract with kit.RPCRouteSelector
type RPCMethod struct {
	Name           string
	Predicate      string
	Request        DTO
	Response       DTO
	PossibleErrors []ErrorDTO
	Encoding       string
	kit.IncomingRPCContainer
	kit.OutgoingRPCContainer
}

func (rm *RPCMethod) addPossibleError(dto ErrorDTO) {
	for _, e := range rm.PossibleErrors {
		if e.Code == dto.Code {
			return
		}
	}
	rm.PossibleErrors = append(rm.PossibleErrors, dto)
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

//nolint:gocognit
func (d *Stub) addDTO(mTyp reflect.Type, isErr bool) error {
	if mTyp.Kind() == reflect.Ptr {
		mTyp = mTyp.Elem()
	}

	dto := DTO{
		Name:  mTyp.Name(),
		Type:  typ("", mTyp),
		IsErr: isErr,
	}

	if mTyp == reflect.TypeOf(kit.RawMessage{}) {
		return nil
	}

	// We don't support non-struct DTOs
	if mTyp.Kind() != reflect.Struct {
		return errUnsupportedType(mTyp.Kind().String())
	}

	// if DTO is already parsed just return
	if _, ok := d.DTOs[dto.Name]; ok {
		return nil
	}

	d.DTOs[dto.Name] = dto

	for i := 0; i < mTyp.NumField(); i++ {
		ft := mTyp.Field(i)
		fe := extractElem(ft)
		dtoF := DTOField{
			Name:     ft.Name,
			Type:     typ("", ft.Type),
			Embedded: ft.Anonymous,
		}

		for _, t := range d.tags {
			v, ok := ft.Tag.Lookup(t)
			if ok {
				dtoF.Tags = append(
					dtoF.Tags,
					DTOFieldTag{
						Name:  t,
						Value: v,
					},
				)
			}
		}

		switch fe.Kind() {
		case reflect.Struct:
			dtoF.IsDTO = true
			err := d.addDTO(fe, false)
			if err != nil {
				return err
			}
		case reflect.Map:
			dtoF.SubType1 = typ("", fe.Key())
			dtoF.SubType2 = typ("", fe.Elem())
			if fe.Key().Kind() == reflect.Struct {
				err := d.addDTO(fe.Key(), false)
				if err != nil {
					return err
				}
			}
			if fe.Elem().Kind() == reflect.Struct {
				err := d.addDTO(fe.Elem(), false)
				if err != nil {
					return err
				}
			}
		case reflect.Slice, reflect.Array:
			dtoF.SubType1 = typ("", fe.Elem())
			if fe.Elem().Kind() == reflect.Struct {
				err := d.addDTO(fe.Elem(), false)
				if err != nil {
					return err
				}
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64, reflect.Bool,
			reflect.String, reflect.Uintptr, reflect.Ptr:
		default:
			continue
		}

		dto.Fields = append(dto.Fields, dtoF)
	}

	d.DTOs[dto.Name] = dto

	return nil
}

func extractElem(in reflect.StructField) reflect.Type {
	t := in.Type
	k := t.Kind()

Loop:
	if k == reflect.Ptr {
		return t.Elem()
	}
	if k == reflect.Slice {
		switch t.Elem().Kind() {
		case reflect.Struct:
			return t.Elem()
		case reflect.Ptr:
			t = t.Elem()
			k = t.Kind()

			goto Loop
		}

		return t.Elem()
	}

	return t
}

func (d *Stub) getDTO(mTyp reflect.Type) (DTO, bool) {
	if mTyp.Kind() == reflect.Ptr {
		mTyp = mTyp.Elem()
	}

	dto, ok := d.DTOs[mTyp.Name()]

	return dto, ok
}

func (d *Stub) Tags() []string {
	return d.tags
}

var errUnsupportedType = errors.NewG("non-struct types as DTO : %s")
