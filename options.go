package ronykit

import log "github.com/ronaksoft/golog"

type Option func(s *EdgeServer)

func WithLogger(l log.Logger) Option {
	return func(s *EdgeServer) {
		s.l = l
	}
}

func RegisterBundle(bundles ...Bundle) Option {
	return func(s *EdgeServer) {
		for _, b := range bundles {
			s.RegisterBundle(b)
		}
	}
}

func RegisterService(services ...Service) Option {
	return func(s *EdgeServer) {
		for _, svc := range services {
			s.RegisterService(svc)
		}
	}
}

func WithErrorHandler(h ErrHandler) Option {
	return func(s *EdgeServer) {
		s.eh = h
	}
}
