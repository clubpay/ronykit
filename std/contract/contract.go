package contract

import "github.com/ronaksoft/ronykit"

// Contract is simple implementation of ronykit.Contract interface.
type Contract struct {
	selectors []ronykit.RouteSelector
	handlers  []ronykit.Handler
}

func New() *Contract {
	return &Contract{}
}

func (r *Contract) SetSelector(selectors ...ronykit.RouteSelector) *Contract {
	r.selectors = append(r.selectors, selectors...)

	return r
}

func (r *Contract) Query(q string) interface{} {
	for _, rd := range r.selectors {
		v := rd.Query(q)
		if v != nil {
			return v
		}
	}

	return nil
}

func (r *Contract) SetHandler(handlers ...ronykit.Handler) *Contract {
	r.handlers = append(r.handlers[:0], handlers...)

	return r
}

func (r *Contract) Handlers() []ronykit.Handler {
	return r.handlers
}
