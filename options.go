package ronykit

import log "github.com/ronaksoft/golog"

type Option func(s *Server)

func WithLogger(l log.Logger) Option {
	return func(s *Server) {
		s.l = l
	}
}

func RegisterBundle(b Bundle) Option {
	return func(s *Server) {
		s.RegisterBundle(b)
	}
}

func RegisterService(srv IService) Option {
	return func(s *Server) {
		s.RegisterService(srv)
	}
}

func WithErrorHandler(h ErrHandler) Option {
	return func(s *Server) {
		s.eh = h
	}
}
