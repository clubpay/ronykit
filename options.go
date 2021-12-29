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
		s.Register(b)
	}
}

func WithErrorHandler(h ErrHandler) Option {
	return func(s *Server) {
		s.eh = h
	}
}
