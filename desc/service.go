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

// AddPossibleError sets the possible errors for this Contract. This is OPTIONAL parameter, which
// mostly could be used by external tools such as Swagger or any other doc generator tools.
func (s *Service) AddPossibleError(code int, item string, m ronykit.Message) *Service {
	s.PossibleErrors = append(s.PossibleErrors, Error{
		Code:    code,
		Item:    item,
		Message: m,
	})

	return s
}

// Generate generates the ronykit.Service
func (s Service) Generate() ronykit.Service {
	svc := &serviceImpl{
		name: s.Name,
		pre:  s.PreHandlers,
		post: s.PostHandlers,
	}

	var index int64 = 0
	for _, c := range s.Contracts {
		reflector.Register(c.Input)
		for _, output := range c.Outputs {
			reflector.Register(output)
		}

		index++

		contracts := make([]ronykit.Contract, len(c.RouteSelectors))
		for idx, s := range c.RouteSelectors {
			ci := (&contractImpl{id: fmt.Sprintf("%s.%d", svc.name, index)}).
				addHandler(svc.pre...).
				addHandler(c.Handlers...).
				addHandler(svc.post...).
				setModifier(c.Modifiers...).
				setInput(c.Input).
				setRouteSelector(s).
				setMemberSelector(c.EdgeSelector).
				setEncoding(c.Encoding)

			contracts[idx] = ronykit.WrapContract(ci, c.Wrappers...)
		}

		svc.contracts = append(svc.contracts, contracts...)
	}

	return ronykit.WrapService(svc, s.Wrappers...)
}

func (s Service) GenerateStub(tags ...string) (*Stub, error) {
	d := &Stub{
		tags: tags,
		DTOs: map[string]DTO{},
	}
	for _, c := range s.Contracts {
		err := d.addDTO(reflect.TypeOf(c.Input))
		if err != nil {
			return nil, err
		}
		for _, out := range c.Outputs {
			err = d.addDTO(reflect.TypeOf(out))
			if err != nil {
				return nil, err
			}
		}

		for _, rs := range c.RouteSelectors {
			if rrs, ok := rs.(ronykit.RESTRouteSelector); ok {
				m := RESTMethod{
					Name:   c.Name,
					Method: rrs.GetMethod(),
					Path:   rrs.GetPath(),
				}
				if dto, ok := d.getDTO(reflect.TypeOf(c.Input)); ok {
					m.Request = dto
				}
				for _, out := range c.Outputs {
					if dto, ok := d.getDTO(reflect.TypeOf(out)); ok {
						m.Response = append(m.Response, dto)
					}
				}
				d.RESTs = append(d.RESTs, m)
			}
		}
	}

	return d, nil
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

func (s *serviceImpl) setPreHandlers(h ...ronykit.HandlerFunc) *serviceImpl {
	s.pre = append(s.pre[:0], h...)

	return s
}

func (s *serviceImpl) setPostHandlers(h ...ronykit.HandlerFunc) *serviceImpl {
	s.post = append(s.post[:0], h...)

	return s
}

func (s *serviceImpl) addContract(contracts ...ronykit.Contract) *serviceImpl {
	s.contracts = append(s.contracts, contracts...)

	return s
}
