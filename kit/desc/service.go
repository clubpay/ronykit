package desc

import (
	"fmt"
	"reflect"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils/reflector"
)

// Service is the description of the kit.Service you are going to create. It then
// generates a kit.Service by calling Generate method.
type Service struct {
	Name           string
	Version        string
	Description    string
	Encoding       kit.Encoding
	PossibleErrors []Error
	Wrappers       []kit.ServiceWrapper
	Contracts      []Contract
	Handlers       []kit.HandlerFunc

	contractNames map[string]struct{}
}

func NewService(name string) *Service {
	return &Service{
		Name:          name,
		contractNames: map[string]struct{}{},
	}
}

func (s *Service) SetEncoding(enc kit.Encoding) *Service {
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

// AddPreHandler Deprecated use AddHandler instead.
func (s *Service) AddPreHandler(h ...kit.HandlerFunc) *Service {
	s.Handlers = append(s.Handlers, h...)

	return s
}

// AddHandler adds handlers to run before and/or after the contract's handlers
func (s *Service) AddHandler(h ...kit.HandlerFunc) *Service {
	s.Handlers = append(s.Handlers, h...)

	return s
}

// AddWrapper adds service wrappers to the Service description.
func (s *Service) AddWrapper(wrappers ...kit.ServiceWrapper) *Service {
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
		if contracts[idx].Encoding == kit.Undefined {
			contracts[idx].Encoding = s.Encoding
		}

		s.Contracts = append(s.Contracts, *contracts[idx])
	}

	return s
}

// AddError sets the possible errors for all the Contracts of this Service.
// Using this method is OPTIONAL, which mostly could be used by external tools such as
// Swagger or any other doc generator tools.
func (s *Service) AddError(err kit.ErrorMessage) *Service {
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

// Generate generates the kit.Service
func (s Service) Generate() kit.Service {
	svc := &serviceImpl{
		name: s.Name,
		h:    s.Handlers,
	}

	var index int64
	for _, c := range s.Contracts {
		reflector.Register(c.Input)
		reflector.Register(c.Output)

		index++
		contractID := fmt.Sprintf("%s.%d", svc.name, index)
		if c.Name != "" {
			if _, ok := s.contractNames[c.Name]; ok {
				panic(fmt.Sprintf("contract name %s already defined in service %s", c.Name, svc.name))
			}
			contractID = fmt.Sprintf("%s.%s", svc.name, c.Name)
			s.contractNames[c.Name] = struct{}{}
		}

		contracts := make([]kit.Contract, len(c.RouteSelectors))
		for idx, s := range c.RouteSelectors {
			ci := (&contractImpl{id: contractID}).
				addHandler(svc.h...).
				addHandler(c.Handlers...).
				setModifier(c.Modifiers...).
				setInput(c.Input).
				setOutput(c.Output).
				setRouteSelector(s.Selector).
				setMemberSelector(c.EdgeSelector).
				setEncoding(c.Encoding)

			contracts[idx] = kit.WrapContract(ci, c.Wrappers...)
		}

		svc.contracts = append(svc.contracts, contracts...)
	}

	return kit.WrapService(svc, s.Wrappers...)
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
			if rrs, ok := rs.Selector.(kit.RESTRouteSelector); ok {
				if rrs.GetMethod() == "" || rrs.GetPath() == "" {
					continue
				}
				s.restStub(stub, c, rs.Name, rrs)
			}

			if rrs, ok := rs.Selector.(kit.RPCRouteSelector); ok {
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
	stub *Stub, c Contract, routeName string, rrs kit.RPCRouteSelector,
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
	stub *Stub, c Contract, routeName string, rrs kit.RESTRouteSelector,
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

func getEncoding(rrs kit.RouteSelector) string {
	switch rrs.GetEncoding().Tag() {
	case kit.JSON.Tag():
		return "kit.ToJSON"
	case kit.Proto.Tag():
		return "kit.Proto"
	case kit.MSG.Tag():
		return "kit.MSG"
	case "":
		return "kit.ToJSON"
	default:
		return fmt.Sprintf("kit.CustomEncoding(%q)", rrs.GetEncoding().Tag())
	}
}

// serviceImpl is a simple implementation of kit.Service interface.
type serviceImpl struct {
	name      string
	h         []kit.HandlerFunc
	contracts []kit.Contract
}

var _ kit.Service = (*serviceImpl)(nil)

func (s *serviceImpl) Name() string {
	return s.name
}

func (s *serviceImpl) Contracts() []kit.Contract {
	return s.contracts
}
