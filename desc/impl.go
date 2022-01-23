package desc

import "github.com/ronaksoft/ronykit"

// serviceImpl is a simple implementation of ronykit.Service interface.
type serviceImpl struct {
	name   string
	pre    []ronykit.Handler
	post   []ronykit.Handler
	routes []ronykit.Contract
}

func (s *serviceImpl) Name() string {
	return s.name
}

func (s *serviceImpl) Contracts() []ronykit.Contract {
	return s.routes
}

func (s *serviceImpl) PreHandlers() []ronykit.Handler {
	return s.pre
}

func (s *serviceImpl) PostHandlers() []ronykit.Handler {
	return s.post
}

func (s *serviceImpl) SetPreHandlers(h ...ronykit.Handler) *serviceImpl {
	s.pre = append(s.pre[:0], h...)

	return s
}

func (s *serviceImpl) SetPostHandlers(h ...ronykit.Handler) *serviceImpl {
	s.post = append(s.post[:0], h...)

	return s
}

func (s *serviceImpl) AddContract(contracts ...ronykit.Contract) *serviceImpl {
	s.routes = append(s.routes, contracts...)

	return s
}

// contractImpl is simple implementation of ronykit.Contract interface.
type contractImpl struct {
	selectors []ronykit.RouteSelector
	handlers  []ronykit.Handler
}

func (r *contractImpl) SetSelector(selectors ...ronykit.RouteSelector) *contractImpl {
	r.selectors = append(r.selectors, selectors...)

	return r
}

func (r *contractImpl) Query(q string) interface{} {
	for _, rd := range r.selectors {
		v := rd.Query(q)
		if v != nil {
			return v
		}
	}

	return nil
}

func (r *contractImpl) SetHandler(handlers ...ronykit.Handler) *contractImpl {
	r.handlers = append(r.handlers[:0], handlers...)

	return r
}

func (r *contractImpl) Handlers() []ronykit.Handler {
	return r.handlers
}
