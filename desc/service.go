package desc

import (
	"fmt"
	"reflect"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/utils/reflector"
)

// Service is the description of the ronykit.Service you are going to create. It then
// generates a ronykit.Service by calling Generate method.
type Service struct {
	Name           string
	Version        string
	Description    string
	Encoding       ronykit.Encoding
	PossibleErrors []Error
	Wrappers       []ronykit.ServiceWrapper
	Contracts      []Contract
	PreHandlers    []ronykit.HandlerFunc
	PostHandlers   []ronykit.HandlerFunc
}

func NewService(name string) *Service {
	return &Service{
		Name: name,
	}
}

func (s *Service) SetEncoding(enc ronykit.Encoding) *Service {
	s.Encoding = enc

	return s
}

func (s *Service) SetVersion(v string) *Service {
	s.Version = v

	return s
}

func (s *Service) SetDescription(d string) *Service {
	s.Description = d

	return s
}

func (s *Service) AddPreHandler(h ...ronykit.HandlerFunc) *Service {
	s.PreHandlers = append(s.PreHandlers, h...)

	return s
}

func (s *Service) AddPostHandler(h ...ronykit.HandlerFunc) *Service {
	s.PostHandlers = append(s.PostHandlers, h...)

	return s
}

// AddWrapper adds service wrappers to the Service description.
func (s *Service) AddWrapper(wrappers ...ronykit.ServiceWrapper) *Service {
	s.Wrappers = append(s.Wrappers, wrappers...)

	return s
}

// AddContract adds a contract to the service.
func (s *Service) AddContract(contracts ...*Contract) *Service {
	for idx := range contracts {
		if contracts[idx] == nil {
			continue
		}

		// If the encoding is not set for contract, it takes the Service encoding as its encoding.
		if contracts[idx].Encoding == ronykit.Undefined {
			contracts[idx].Encoding = s.Encoding
		}

		s.Contracts = append(s.Contracts, *contracts[idx])
	}

	return s
}

// AddError sets the possible errors for all the Contracts of this Service.
// Using this method is OPTIONAL, which mostly could be used by external tools such as
// Swagger or any other doc generator tools.
func (s *Service) AddError(err ronykit.ErrorMessage) *Service {
	s.PossibleErrors = append(
		s.PossibleErrors,
		Error{
			Code:    err.GetCode(),
			Item:    err.GetItem(),
			Message: err,
		},
	)

	return s
}

// Generate generates the ronykit.Service
func (s Service) Generate() ronykit.Service {
	svc := &serviceImpl{
		name: s.Name,
		pre:  s.PreHandlers,
		post: s.PostHandlers,
	}

	var index int64
	for _, c := range s.Contracts {
		reflector.Register(c.Input)
		reflector.Register(c.Output)

		index++

		contracts := make([]ronykit.Contract, len(c.RouteSelectors))
		for idx, s := range c.RouteSelectors {
			ci := (&contractImpl{id: fmt.Sprintf("%s.%d", svc.name, index)}).
				addHandler(svc.pre...).
				addHandler(c.Handlers...).
				addHandler(svc.post...).
				setModifier(c.Modifiers...).
				setInput(c.Input).
				setRouteSelector(s.Selector).
				setMemberSelector(c.EdgeSelector).
				setEncoding(c.Encoding)

			contracts[idx] = ronykit.WrapContract(ci, c.Wrappers...)
		}

		svc.contracts = append(svc.contracts, contracts...)
	}

	return ronykit.WrapService(svc, s.Wrappers...)
}

// Stub returns the Stub, which describes the stub specification and
// could be used to auto-generate stub for this service.
func (s Service) Stub(pkgName string, tags ...string) (*Stub, error) {
	stub := newStub(tags...)
	stub.Pkg = pkgName
	stub.Name = s.Name

	if err := s.dtoStub(stub); err != nil {
		return nil, err
	}

	for _, c := range s.Contracts {
		for _, rs := range c.RouteSelectors {
			if rrs, ok := rs.Selector.(ronykit.RESTRouteSelector); ok {
				if rrs.GetMethod() == "" || rrs.GetPath() == "" {
					continue
				}
				s.restStub(stub, c, rs.Name, rrs)
			}

			if rrs, ok := rs.Selector.(ronykit.RPCRouteSelector); ok {
				s.rpcStub(stub, c, rs.Name, rrs)
			}
		}
	}

	return stub, nil
}

func (s Service) dtoStub(stub *Stub) error {
	for _, c := range s.Contracts {
		err := stub.addDTO(reflect.TypeOf(c.Input), false)
		if err != nil {
			return err
		}
		if c.Output != nil {
			err = stub.addDTO(reflect.TypeOf(c.Output), false)
			if err != nil {
				return err
			}
		}

		for _, pe := range s.PossibleErrors {
			err = stub.addDTO(reflect.TypeOf(pe.Message), true)
			if err != nil {
				return err
			}
		}

		for _, pe := range c.PossibleErrors {
			err = stub.addDTO(reflect.TypeOf(pe.Message), true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s Service) rpcStub(
	stub *Stub, c Contract, routeName string, rrs ronykit.RPCRouteSelector,
) {
	m := RPCMethod{
		Name:      routeName,
		Predicate: rrs.GetPredicate(),
		Encoding:  getEncoding(rrs),
	}

	if dto, ok := stub.getDTO(reflect.TypeOf(c.Input)); ok {
		m.Request = dto
	}
	if c.Output != nil {
		if dto, ok := stub.getDTO(reflect.TypeOf(c.Output)); ok {
			m.Response = dto
		}
	}

	var possibleErrors []Error
	possibleErrors = append(possibleErrors, s.PossibleErrors...)
	possibleErrors = append(possibleErrors, c.PossibleErrors...)
	for _, e := range possibleErrors {
		if dto, ok := stub.getDTO(reflect.TypeOf(e.Message)); ok {
			m.addPossibleError(
				ErrorDTO{
					Code: e.Code,
					Item: e.Item,
					DTO:  dto,
				},
			)
		}
	}
}

func (s Service) restStub(
	stub *Stub, c Contract, routeName string, rrs ronykit.RESTRouteSelector,
) {
	m := RESTMethod{
		Name:     routeName,
		Method:   rrs.GetMethod(),
		Path:     rrs.GetPath(),
		Encoding: getEncoding(rrs),
	}

	if dto, ok := stub.getDTO(reflect.TypeOf(c.Input)); ok {
		m.Request = dto
	}
	if c.Output != nil {
		if dto, ok := stub.getDTO(reflect.TypeOf(c.Output)); ok {
			m.Response = dto
		}
	}

	var possibleErrors []Error
	possibleErrors = append(possibleErrors, s.PossibleErrors...)
	possibleErrors = append(possibleErrors, c.PossibleErrors...)
	for _, e := range possibleErrors {
		if dto, ok := stub.getDTO(reflect.TypeOf(e.Message)); ok {
			m.addPossibleError(
				ErrorDTO{
					Code: e.Code,
					Item: e.Item,
					DTO:  dto,
				},
			)
		}
	}

	stub.RESTs = append(stub.RESTs, m)
}

func getEncoding(rrs ronykit.RouteSelector) string {
	switch rrs.GetEncoding().Tag() {
	case ronykit.JSON.Tag():
		return "ronykit.JSON"
	case ronykit.Proto.Tag():
		return "ronykit.Proto"
	case ronykit.MSG.Tag():
		return "ronykit.MSG"
	case "":
		return "ronykit.JSON"
	default:
		return fmt.Sprintf("ronykit.CustomEncoding(%q)", rrs.GetEncoding().Tag())
	}
}

// serviceImpl is a simple implementation of ronykit.Service interface.
type serviceImpl struct {
	name      string
	pre       []ronykit.HandlerFunc
	post      []ronykit.HandlerFunc
	contracts []ronykit.Contract
}

var _ ronykit.Service = (*serviceImpl)(nil)

func (s *serviceImpl) Name() string {
	return s.name
}

func (s *serviceImpl) Contracts() []ronykit.Contract {
	return s.contracts
}
