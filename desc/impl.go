package desc

import "github.com/clubpay/ronykit"

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

// contractImpl is simple implementation of ronykit.Contract interface.
type contractImpl struct {
	id             string
	routeSelector  ronykit.RouteSelector
	memberSelector ronykit.EdgeSelector
	handlers       []ronykit.HandlerFunc
	modifiers      []ronykit.Modifier
	input          ronykit.Message
	enc            ronykit.Encoding
}

var _ ronykit.Contract = (*contractImpl)(nil)

func (r *contractImpl) setInput(input ronykit.Message) *contractImpl {
	r.input = input

	return r
}

func (r *contractImpl) setEncoding(enc ronykit.Encoding) *contractImpl {
	r.enc = enc

	return r
}

func (r *contractImpl) setRouteSelector(selector ronykit.RouteSelector) *contractImpl {
	r.routeSelector = selector

	return r
}

func (r *contractImpl) setMemberSelector(selector ronykit.EdgeSelector) *contractImpl {
	r.memberSelector = selector

	return r
}

func (r *contractImpl) setHandler(handlers ...ronykit.HandlerFunc) *contractImpl {
	r.handlers = append(r.handlers[:0], handlers...)

	return r
}

func (r *contractImpl) addHandler(handlers ...ronykit.HandlerFunc) *contractImpl {
	r.handlers = append(r.handlers, handlers...)

	return r
}

func (r *contractImpl) setModifier(modifiers ...ronykit.Modifier) *contractImpl {
	r.modifiers = append(r.modifiers[:0], modifiers...)

	return r
}

func (r *contractImpl) ID() string {
	return r.id
}

func (r *contractImpl) RouteSelector() ronykit.RouteSelector {
	return r.routeSelector
}

func (r *contractImpl) EdgeSelector() ronykit.EdgeSelector {
	return r.memberSelector
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
