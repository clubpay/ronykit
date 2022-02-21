package desc

import (
	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/reflector"
)

type Error struct {
	Code    int
	Item    string
	Message ronykit.Message
}

// Contract is the description of the ronykit.Contract you are going to create.
type Contract struct {
	Name           string
	Encoding       ronykit.Encoding
	Handlers       []ronykit.HandlerFunc
	Wrappers       []ronykit.ContractWrapper
	Input          ronykit.Message
	Output         ronykit.Message
	PossibleErrors []Error
	Selectors      []ronykit.RouteSelector
	Modifiers      []ronykit.Modifier
}

func NewContract() *Contract {
	return &Contract{}
}

// SetName sets the name of the Contract c, it MUST be unique per Service. However, it
// has no operation effect only is helpful with some other tools such as monitoring,
// logging or tracing tools to identity it.
func (c *Contract) SetName(name string) *Contract {
	c.Name = name

	return c
}

// SetEncoding sets the supported encoding for this contract.
func (c *Contract) SetEncoding(enc ronykit.Encoding) *Contract {
	c.Encoding = enc

	return c
}

// SetInput sets the accepting message for this Contract. Contracts are bound to one input message.
// In some odd cases if you need to handle multiple input messages, then you SHOULD create multiple
// contracts but with same handlers and/or selectors.
func (c *Contract) SetInput(m ronykit.Message) *Contract {
	c.Input = m

	return c
}

// SetOutput sets the outgoing message for this Contract. This is an OPTIONAL parameter, which
// mostly could be used by external tools such as Swagger or any other doc generator tools.
func (c *Contract) SetOutput(m ronykit.Message) *Contract {
	c.Output = m

	return c
}

// AddPossibleError sets the possible errors for this Contract. This is OPTIONAL parameter, which
// mostly could be used by external tools such as Swagger or any other doc generator tools.
func (c *Contract) AddPossibleError(code int, item string, m ronykit.Message) *Contract {
	c.PossibleErrors = append(c.PossibleErrors, Error{
		Code:    code,
		Item:    item,
		Message: m,
	})

	return c
}

// AddSelector adds a ronykit.RouteSelector for this contract. Selectors are bundle specific.
func (c *Contract) AddSelector(s ronykit.RouteSelector) *Contract {
	c.Selectors = append(c.Selectors, s)

	return c
}

// AddModifier adds a ronykit.Modifier for this contract. Modifiers are used to modify
// the outgoing ronykit.Envelope just before sending to the client.
func (c *Contract) AddModifier(m ronykit.Modifier) *Contract {
	c.Modifiers = append(c.Modifiers, m)

	return c
}

// AddWrapper adds a ronykit.ContractWrapper for this contract.
func (c *Contract) AddWrapper(wrappers ...ronykit.ContractWrapper) *Contract {
	c.Wrappers = append(c.Wrappers, wrappers...)

	return c
}

// AddHandler add handler for this contract.
func (c *Contract) AddHandler(h ...ronykit.HandlerFunc) *Contract {
	c.Handlers = append(c.Handlers, h...)

	return c
}

// SetHandler set the handler by replacing the already existing ones.
func (c *Contract) SetHandler(h ...ronykit.HandlerFunc) *Contract {
	c.Handlers = append(c.Handlers[:0], h...)

	return c
}

// Generate generates the ronykit.Contract
func (c *Contract) Generate() []ronykit.Contract {
	//nolint:prealloc
	var contracts []ronykit.Contract

	for _, s := range c.Selectors {
		ci := &contractImpl{}
		ci.setHandler(c.Handlers...)
		ci.setModifier(c.Modifiers...)
		ci.setInput(c.Input)
		ci.setSelector(s)
		ci.setEncoding(c.Encoding)

		contracts = append(contracts, ronykit.WrapContract(ci, c.Wrappers...))
	}

	return contracts
}

// Service is the description of the ronykit.Service you are going to create. It then
// generates a ronykit.Service by calling Generate method.
type Service struct {
	Name         string
	Version      string
	Description  string
	Encoding     ronykit.Encoding
	Wrappers     []ronykit.ServiceWrapper
	Contracts    []Contract
	PreHandlers  []ronykit.HandlerFunc
	PostHandlers []ronykit.HandlerFunc
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

// Add adds a contract to the service.
// Deprecated use AddContract instead.
func (s *Service) Add(c *Contract) *Service {
	s.AddContract(c)

	return s
}

// Generate generates the ronykit.Service
func (s Service) Generate() ronykit.Service {
	svc := &serviceImpl{
		name: s.Name,
		pre:  s.PreHandlers,
		post: s.PostHandlers,
	}

	for _, c := range s.Contracts {
		reflector.Register(c.Input)
		reflector.Register(c.Output)
		svc.contracts = append(svc.contracts, c.Generate()...)
	}

	return ronykit.WrapService(svc, s.Wrappers...)
}
