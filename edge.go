package ronykit

import (
	"os"
	"os/signal"
	"sync"

	"github.com/ronaksoft/ronykit/log"
)

type (
	Handler func(ctx *Context) Handler
	Bundle  interface {
		Gateway() Gateway
		Dispatcher() Dispatcher
	}
)

type Server struct {
	nb []*northBridge
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

func (s *Server) Register(b Bundle) {
	nb := &northBridge{
		ctxPool: sync.Pool{},
		l:       s.l,
		gw:      b.Gateway(),
		d:       b.Dispatcher(),
	}
	s.nb = append(s.nb, nb)

	b.Gateway().Subscribe(nb)
}

func (s *Server) Start() {
	// Start all the registered gateways
	for idx := range s.nb {
		s.nb[idx].gw.Start()
	}
	s.l.Debug("server started.")
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
}
