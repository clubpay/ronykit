package desc

import "github.com/ronaksoft/ronykit"

// Service is the description of the ronykit.Service you are going to create. It then
// generates a ronykit.Service by calling Generate method.
type Service struct {
	Name      string
	Contracts []Contract
}

func (s *Service) Add(c Contract) *Service {
	s.Contracts = append(s.Contracts, c)

	return s
}

func (s Service) Generate() ronykit.Service {

	return nil
}

type Contract struct {
	Name      string
	Handlers  []ronykit.Handler
	Input     ronykit.Message
	RESTs     []REST
	Predicate string
}

func (c *Contract) SetName(name string) *Contract {
	c.Name = name

	return c
}

func (c *Contract) AddInput(m ronykit.Message) *Contract {
	c.Input = m

	return c
}

func (c *Contract) AddREST(r REST) *Contract {
	c.RESTs = append(c.RESTs, r)

	return c
}

func (c *Contract) WithHandlers(h ...ronykit.Handler) *Contract {
	c.Handlers = append(c.Handlers, h...)

	return c
}

func (c *Contract) WithPredicate(p string) *Contract {
	c.Predicate = p
	
	return c
}

type REST struct {
	Method string
	Path   string
}
