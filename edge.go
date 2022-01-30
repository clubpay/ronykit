package ronykit

import (
	"fmt"
	"os"
	"os/signal"

	log "github.com/ronaksoft/golog"
)

type (
	ErrHandler func(ctx *Context, err error)
	Handler    func(ctx *Context)
	Bundle     interface {
		Gateway
		Dispatcher
		Register(svc Service)
	}
)

type Server struct {
	nb  []*northBridge
	svc map[string]Service
	eh  ErrHandler
	l   log.Logger
}

func NewServer(opts ...Option) *Server {
	s := &Server{
		l:   log.NopLogger,
		svc: map[string]Service{},
	}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Server) RegisterBundle(b Bundle) *Server {
	nb := &northBridge{
		l:  s.l,
		b:  b,
		eh: s.eh,
	}
	s.nb = append(s.nb, nb)

	b.Subscribe(nb)

	return s
}

func (s *Server) RegisterService(svc Service) *Server {
	if _, ok := s.svc[svc.Name()]; ok {
		panic(fmt.Sprintf("service %s already registered", svc.Name()))
	}

	s.svc[svc.Name()] = svc

	return s
}

func (s *Server) Start() *Server {
	// Register services into the bundles and start them.
	for idx := range s.nb {
		for _, svc := range s.svc {
			s.nb[idx].b.Register(svc)
		}

		s.nb[idx].b.Start()
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
		s.nb[idx].b.Shutdown()
	}

	return
}
