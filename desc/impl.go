package desc

import "github.com/clubpay/ronykit"

// serviceImpl is a simple implementation of ronykit.Service interface.
type serviceImpl struct {
	name      string
	pre       []ronykit.HandlerFunc
	post      []ronykit.HandlerFunc
	contracts []ronykit.Contract
}

func (s *serviceImpl) Name() string {
	return s.name
}

func (s *serviceImpl) Contracts() []ronykit.Contract {
	return s.contracts
}

func (s *serviceImpl) PreHandlers() []ronykit.HandlerFunc {
	return s.pre
}

func (s *serviceImpl) PostHandlers() []ronykit.HandlerFunc {
	return s.post
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

// contractImpl is simple implementation of ronykit.Contract interface.
type contractImpl struct {
	selector  ronykit.RouteSelector
	handlers  []ronykit.HandlerFunc
	modifiers []ronykit.Modifier
	input     ronykit.Message
	enc       ronykit.Encoding
}

func (r *contractImpl) setInput(input ronykit.Message) *contractImpl {
	r.input = input

	return r
}

func (r *contractImpl) setEncoding(enc ronykit.Encoding) *contractImpl {
	r.enc = enc

	return r
}

func (r *contractImpl) setSelector(selector ronykit.RouteSelector) *contractImpl {
	r.selector = selector

	return r
}

func (r *contractImpl) setHandler(handlers ...ronykit.HandlerFunc) *contractImpl {
	r.handlers = append(r.handlers[:0], handlers...)

	return r
}

func (r *contractImpl) setModifier(modifiers ...ronykit.Modifier) *contractImpl {
	r.modifiers = append(r.modifiers[:0], modifiers...)

	return r
}

func (r *contractImpl) Selector() ronykit.RouteSelector {
	return r.selector
}

func (r *contractImpl) Handlers() []ronykit.HandlerFunc {
	return r.handlers
}

func (r *contractImpl) Modifiers() []ronykit.Modifier {
	return r.modifiers
}

func (r *contractImpl) Input() ronykit.Message {
	return r.input
}

func (r *contractImpl) Encoding() ronykit.Encoding {
	return r.enc
}
