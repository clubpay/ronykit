package desc

import (
	"fmt"

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

var _ kit.ServiceBuilder = (*Service)(nil)

func NewService(name string) *Service {
	return &Service{
		Name:          name,
		Encoding:      kit.JSON,
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
// NOTE:	The auto-generated stub also use these errors to identifies if the response should be considered
//
//	as error or successful.
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

// Build generates the kit.Service
func (s *Service) Build() kit.Service {
	svc := &serviceImpl{
		name: s.Name,
		h:    s.Handlers,
	}

	var index int64
	for _, c := range s.Contracts {
		reflector.Register(c.Input, s.Encoding.Tag(), c.Encoding.Tag())
		reflector.Register(c.Output, s.Encoding.Tag(), c.Encoding.Tag())

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
