package edge

import (
	"os"
	"os/signal"
	"sync"

	"github.com/ronaksoft/ronykit"
)

type Handler func(ctx *RequestCtx, envelope ronykit.Envelope) Handler

type Server struct {
	nb []*northBridge
}

func New() *Server {
	s := &Server{}

	return s
}

func (s *Server) RegisterGateway(
	gw ronykit.Gateway, b Bundle,
) {
	nb := &northBridge{
		ctxPool: sync.Pool{},
		r:       b.Router,
		ep:      b.EnvelopePool,
		gw:      gw,
		d:       b.Dispatcher,
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
