package ronykit

import (
	"os"
	"os/signal"
	"sync"

	"github.com/ronaksoft/ronykit/log"
)

type (
	ErrHandler func(err error)
	Handler    func(ctx *Context) Handler
	Bundle     interface {
		Gateway
		Dispatcher
	}
)

type Server struct {
	nb []*northBridge
	eh ErrHandler
	l  log.Logger
}

func NewServer(opts ...Option) *Server {
	s := &Server{
		l: log.NopLogger,
	}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Server) Register(b Bundle) *Server {
	nb := &northBridge{
		ctxPool: sync.Pool{},
		l:       s.l,
		gw:      b,
		d:       b,
		eh:      s.eh,
	}
	s.nb = append(s.nb, nb)

	b.Subscribe(nb)

	return s
}

func (s *Server) Start() *Server {
	// Start all the registered gateways
	for idx := range s.nb {
		s.nb[idx].gw.Start()
	}
	s.l.Debug("server started.")

	return s
}

func (s *Server) Shutdown(signals ...os.Signal) {
	if len(signals) > 0 {
		// Create a signal channel and bind it to all the os signals in the arg
		shutdownChan := make(chan os.Signal, 1)
		signal.Notify(shutdownChan, signals...)

		// Wait for the shutdown signal
		<-shutdownChan
	}

	// Start all the registered gateways
	for idx := range s.nb {
		s.nb[idx].gw.Shutdown()
	}

	return
}
