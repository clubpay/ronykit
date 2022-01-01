package ronykit

import (
	"fmt"
	"os"
	"os/signal"
	"sync"

	log "github.com/ronaksoft/golog"
)

type (
	ErrHandler func(err error)
	Handler    func(ctx *Context) Handler
	Bundle     interface {
		Gateway
		Dispatcher
		Register(srv IService)
	}
)

type Server struct {
	nb  []*northBridge
	srv map[string]IService
	eh  ErrHandler
	l   log.Logger
}

func NewServer(opts ...Option) *Server {
	s := &Server{
		l:   log.NopLogger,
		srv: map[string]IService{},
	}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Server) RegisterBundle(b Bundle) *Server {
	nb := &northBridge{
		ctxPool: sync.Pool{},
		l:       s.l,
		b:       b,
		eh:      s.eh,
	}
	s.nb = append(s.nb, nb)

	b.Subscribe(nb)

	return s
}

func (s *Server) RegisterService(srv IService) *Server {
	if _, ok := s.srv[srv.Name()]; ok {
		panic(fmt.Sprintf("service %s already registered", srv.Name()))
	}

	s.srv[srv.Name()] = srv

	return s
}

func (s *Server) Start() *Server {
	// Register services into the bundles and start them.
	for idx := range s.nb {
		for _, srv := range s.srv {
			s.nb[idx].b.Register(srv)
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
