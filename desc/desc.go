package desc

import (
	"reflect"

	"github.com/ronaksoft/ronykit"
)

// Selector is the interface which should be provided by the Bundle developer. That is we
// make it as interface instead of concrete implementation.
type Selector interface {
	Generate(f ronykit.MessageFactory) ronykit.RouteSelector
}

// Contract is the description of the ronykit.Contract you are going to create.
type Contract struct {
	Name      string
	Handlers  []ronykit.Handler
	Input     ronykit.Message
	Output    ronykit.Message
	Selectors []Selector
	Modifiers []ronykit.Modifier
}

func NewContract() *Contract {
	return &Contract{}
}

func (c *Contract) SetName(name string) *Contract {
	c.Name = name

	return c
}

func (c *Contract) SetInput(m ronykit.Message) *Contract {
	c.Input = m

	return c
}

func (c *Contract) SetOutput(m ronykit.Message) *Contract {
	c.Output = m

	return c
}

func (c *Contract) AddSelector(s Selector) *Contract {
	c.Selectors = append(c.Selectors, s)

	return c
}

func (c *Contract) AddModifier(m ronykit.Modifier) *Contract {
	c.Modifiers = append(c.Modifiers, m)

	return c
}

func (c *Contract) AddHandler(h ...ronykit.Handler) *Contract {
	c.Handlers = append(c.Handlers, h...)

	return c
}

func (c *Contract) SetHandler(h ...ronykit.Handler) *Contract {
	c.Handlers = append(c.Handlers[:0], h...)

	return c
}

func (c *Contract) Generate() []ronykit.Contract {
	//nolint:prealloc
	var contracts []ronykit.Contract
	makeFunc := func(m ronykit.Message) func(args []reflect.Value) (results []reflect.Value) {
		return func(args []reflect.Value) (results []reflect.Value) {
			return []reflect.Value{reflect.New(reflect.TypeOf(m).Elem())}
		}
	}

	for _, s := range c.Selectors {
		ci := &contractImpl{}
		ci.setHandler(c.Handlers...)
		ci.setModifier(c.Modifiers...)

		reflect.ValueOf(&ci.factoryFunc).Elem().Set(
			reflect.MakeFunc(
				reflect.TypeOf(ci.factoryFunc),
				makeFunc(c.Input),
			),
		)

		ci.setSelector(s.Generate(ci.factoryFunc))

		contracts = append(contracts, ci)
	}

	return contracts
}

// Service is the description of the ronykit.Service you are going to create. It then
// generates a ronykit.Service by calling Generate method.
type Service struct {
	Name         string
	Version      string
	Description  string
	Contracts    []Contract
	PreHandlers  []ronykit.Handler
	PostHandlers []ronykit.Handler
}

func (s *Service) Add(c *Contract) *Service {
	if c != nil {
		s.Contracts = append(s.Contracts, *c)
	}

	return s
}

func (s Service) Generate() ronykit.Service {
	svc := &serviceImpl{
		name: s.Name,
		pre:  s.PreHandlers,
		post: s.PostHandlers,
	}

	for _, c := range s.Contracts {
		svc.contracts = append(svc.contracts, c.Generate()...)
	}

	return svc
}
