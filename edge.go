package ronykit

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/clubpay/ronykit/internal/errors"
)

var errServiceAlreadyRegistered errors.ErrFunc = func(v ...interface{}) error {
	return fmt.Errorf("service %s already registered", v...)
}

// EdgeServer is the main component of the ronykit. It glues all other components of the
// app to each other.
type EdgeServer struct {
	nb  []*northBridge
	sb  []*southBridge
	svc map[string]Service
	eh  ErrHandler
	l   Logger
}

func NewServer(opts ...Option) *EdgeServer {
	s := &EdgeServer{
		l:   nopLogger{},
		svc: map[string]Service{},
		eh:  func(ctx *Context, err error) {},
	}
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// RegisterGateway registers a Gateway to our server. EdgeServer needs at least one Gateway
// to be registered.
func (s *EdgeServer) RegisterGateway(b Gateway) *EdgeServer {
	nb := &northBridge{
		l:  s.l,
		b:  b,
		eh: s.eh,
	}
	s.nb = append(s.nb, nb)

	// Subscribe the northBridge, which is a GatewayDelegate, to connect northBridge with the Gateway
	b.Subscribe(nb)

	return s
}

// RegisterCluster register a Cluster to our server. Registering Cluster is an optional
// feature, but setting up cluster for our EdgeServer it provides capabilities to let
// communication between different instances of the same EdgeServer.
func (s *EdgeServer) RegisterCluster(c Cluster) *EdgeServer {
	sb := &southBridge{
		l:  s.l,
		c:  c,
		eh: s.eh,
	}
	s.sb = append(s.sb, sb)

	// Subscribe the southBridge, which is a ClusterDelegate, to connect southBridge with the Cluster
	c.Subscribe(sb)

	return s
}

// RegisterService registers a Service to our server. We need to define the appropriate
// RouteSelector in each desc.Contract.
func (s *EdgeServer) RegisterService(svc Service) *EdgeServer {
	if _, ok := s.svc[svc.Name()]; ok {
		panic(errServiceAlreadyRegistered(svc.Name()))
	}

	s.svc[svc.Name()] = WrapServiceContracts(svc, ContractWrapperFunc(wrapWithForwarder))

	return s
}

// Start registers services in the registered bundles and start the bundles.
func (s *EdgeServer) Start() *EdgeServer {
	for idx := range s.nb {
		for _, svc := range s.svc {
			s.nb[idx].b.Register(svc)
		}

		s.nb[idx].b.Start()
	}

	s.l.Debug("server started.")

	return s
}

func (s *EdgeServer) Shutdown(signals ...os.Signal) {
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
