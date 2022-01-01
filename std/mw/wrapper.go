package mw

import "github.com/ronaksoft/ronykit"

type serviceWrap struct {
	srv  ronykit.IService
	pre  ronykit.Handler
	post ronykit.Handler
}

func (s serviceWrap) Name() string {
	return s.srv.Name()
}

func (s serviceWrap) Routes() []ronykit.IRoute {
	return s.srv.Routes()
}

func (s serviceWrap) PreHandlers() []ronykit.Handler {
	var handlers = []ronykit.Handler{s.pre}

	return append(handlers, s.srv.PreHandlers()...)
}

func (s serviceWrap) PostHandlers() []ronykit.Handler {
	var handlers = []ronykit.Handler{s.post}

	return append(handlers, s.srv.PostHandlers()...)
}
