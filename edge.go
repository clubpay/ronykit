package ronykit

import (
	"github.com/ronaksoft/ronykit/log"
	"os"
	"os/signal"
	"sync"
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
	s := &Server{}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Server) Register(b Bundle) {
	nb := &northBridge{
		ctxPool: sync.Pool{},
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
