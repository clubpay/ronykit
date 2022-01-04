package mw

import "github.com/ronaksoft/ronykit"

type serviceWrap struct {
	svc  ronykit.Service
	pre  ronykit.Handler
	post ronykit.Handler
}

func Wrap(svc ronykit.Service, pre, post ronykit.Handler) *serviceWrap {
	return &serviceWrap{
		svc:  svc,
		pre:  pre,
		post: post,
	}
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
	return append(s.svc.PostHandlers(), s.post)
}
