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

func (s *serviceImpl) setPreHandlers(h ...ronykit.Handler) *serviceImpl {
	s.pre = append(s.pre[:0], h...)

	return s
}

func (s *serviceImpl) setPostHandlers(h ...ronykit.Handler) *serviceImpl {
	s.post = append(s.post[:0], h...)

	return s
}

func (s *serviceImpl) addContract(contracts ...ronykit.Contract) *serviceImpl {
	s.routes = append(s.routes, contracts...)

	return s
}

// contractImpl is simple implementation of ronykit.Contract interface.
type contractImpl struct {
	selectors []ronykit.RouteSelector
	handlers  []ronykit.Handler
}

func (r *contractImpl) addSelector(selectors ...ronykit.RouteSelector) *contractImpl {
	r.selectors = append(r.selectors, selectors...)

	return r
}

func (r *contractImpl) setHandler(handlers ...ronykit.Handler) *contractImpl {
	r.handlers = append(r.handlers[:0], handlers...)

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

func (r *contractImpl) Handlers() []ronykit.Handler {
	return r.handlers
}
