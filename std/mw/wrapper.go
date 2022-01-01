package mw

import "github.com/ronaksoft/ronykit"

type serviceWrap struct {
	svc  ronykit.Service
	pre  ronykit.Handler
	post ronykit.Handler
}

func (s serviceWrap) Name() string {
	return s.svc.Name()
}

func (s serviceWrap) Routes() []ronykit.Route {
	return s.svc.Routes()
}

func (s serviceWrap) PreHandlers() []ronykit.Handler {
	var handlers = []ronykit.Handler{s.pre}

	return append(handlers, s.svc.PreHandlers()...)
}

func (s serviceWrap) PostHandlers() []ronykit.Handler {
	var handlers = []ronykit.Handler{s.post}

	return append(handlers, s.svc.PostHandlers()...)
}
