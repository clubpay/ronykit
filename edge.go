package ronykit

import (
	"os"
	"os/signal"
	"sync"
)

type Handler func(ctx *Context) Handler

type Server struct {
	nb []*northBridge
}

func NewServer() *Server {
	s := &Server{}

	return s
}

func (s *Server) RegisterGateway(gw Gateway, d Dispatcher) {
	nb := &northBridge{
		ctxPool: sync.Pool{},
		gw:      gw,
		d:       d,
	}
	s.nb = append(s.nb, nb)

	gw.Subscribe(nb)
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
