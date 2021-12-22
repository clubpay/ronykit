package edge

import (
	"os"
	"os/signal"
	"sync"

	"github.com/ronaksoft/ronykit"
)

type Handler func(ctx *RequestCtx, envelope ronykit.Envelope) Handler

type Server struct {
	ep ronykit.EnvelopePool
	r  Router
	nb []*northBridge
}

func New(r Router, ep ronykit.EnvelopePool) *Server {
	s := &Server{
		r:  r,
		ep: ep,
	}

	return s
}

func (s *Server) RegisterGateway(
	gw ronykit.Gateway, d ronykit.Dispatcher,
) {
	nb := &northBridge{
		ctxPool: sync.Pool{},
		r:       s.r,
		ep:      s.ep,
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
