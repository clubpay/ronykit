package desc

import "github.com/ronaksoft/ronykit"

// serviceImpl is a simple implementation of ronykit.Service interface.
type serviceImpl struct {
	name      string
	pre       []ronykit.Handler
	post      []ronykit.Handler
	contracts []ronykit.Contract
}

func (s *serviceImpl) Name() string {
	return s.name
}

func (s *serviceImpl) Contracts() []ronykit.Contract {
	return s.contracts
}

func (s *serviceImpl) PreHandlers() []ronykit.Handler {
	return s.pre
}

func (s *serviceImpl) PostHandlers() []ronykit.Handler {
	return s.post
}

func (s *serviceImpl) setPreHandlers(h ...ronykit.Handler) *serviceImpl {
	s.pre = append(s.pre[:0], h...)

	return s
}

func (s *serviceImpl) setPostHandlers(h ...ronykit.Handler) *serviceImpl {
	s.post = append(s.post[:0], h...)

	return s
}

func (s *serviceImpl) addContract(contracts ...ronykit.Contract) *serviceImpl {
	s.contracts = append(s.contracts, contracts...)

	return s
}

// contractImpl is simple implementation of ronykit.Contract interface.
type contractImpl struct {
	selector  ronykit.RouteSelector
	handlers  []ronykit.Handler
	modifiers []ronykit.Modifier
	input     ronykit.Message
}

func (r *contractImpl) setInput(input ronykit.Message) *contractImpl {
	r.input = input

	return r
}

func (r *contractImpl) setSelector(selector ronykit.RouteSelector) *contractImpl {
	r.selector = selector

	return r
}

func (r *contractImpl) setHandler(handlers ...ronykit.Handler) *contractImpl {
	r.handlers = append(r.handlers[:0], handlers...)

	return r
}

func (r *contractImpl) setModifier(modifiers ...ronykit.Modifier) *contractImpl {
	r.modifiers = append(r.modifiers[:0], modifiers...)

	return r
}

func (r *contractImpl) Query(q string) interface{} {
	return r.selector.Query(q)
}

func (r *contractImpl) Handlers() []ronykit.Handler {
	return r.handlers
}

func (r *contractImpl) Modifiers() []ronykit.Modifier {
	return r.modifiers
}

func (r *contractImpl) Input() ronykit.Message {
	return r.input
}
