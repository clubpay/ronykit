package kit

import "time"

type edgeConfig struct {
	logger          Logger
	prefork         bool
	shutdownTimeout time.Duration
	gateways        []Gateway
	cluster         Cluster
	services        []Service
	errHandler      ErrHandlerFunc
	globalHandlers  []HandlerFunc
	tracer          Tracer
	connDelegate    ConnDelegate
}

type Option func(s *edgeConfig)

func WithLogger(l Logger) Option {
	return func(s *edgeConfig) {
		s.logger = l
	}
}

func WithPrefork() Option {
	return func(s *edgeConfig) {
		s.prefork = true
	}
}

// WithGateway lets you register a bundle in constructor of the EdgeServer.
func WithGateway(gw ...Gateway) Option {
	return func(s *edgeConfig) {
		s.gateways = append(s.gateways, gw...)
	}
}

// WithCluster lets you register a cluster in constructor of the EdgeServer.
func WithCluster(cb Cluster) Option {
	return func(s *edgeConfig) {
		s.cluster = cb
	}
}

// WithService lets you register a service in constructor of the EdgeServer.
func WithService(service ...Service) Option {
	return func(s *edgeConfig) {
		s.services = append(s.services, service...)
	}
}

// WithServiceDesc lets you register a ServiceDescriptor in constructor of the EdgeServer.
func WithServiceDesc(desc ...ServiceDescriptor) Option {
	return func(s *edgeConfig) {
		for _, d := range desc {
			s.services = append(s.services, d.Generate())
		}
	}
}

// WithShutdownTimeout sets the maximum time to wait until all running requests to finish.
// Default is to wait forever.
func WithShutdownTimeout(d time.Duration) Option {
	return func(s *edgeConfig) {
		s.shutdownTimeout = d
	}
}

// WithErrorHandler registers a global error handler to catch any error that
// happens before EdgeServer can deliver the incoming message to the handler, or delivering
// the outgoing message to the client.
// Internal errors are usually wrapped and could be checked for better error handling.
// You can check with errors.Is function to see if the error is one of the following:
// ErrDispatchFailed, ErrWriteToClosedConn, ErrNoHandler
// ErrDecodeIncomingMessageFailed, ErrEncodeOutgoingMessageFailed
func WithErrorHandler(h ErrHandlerFunc) Option {
	return func(s *edgeConfig) {
		s.errHandler = h
	}
}

// WithGlobalHandlers sets the handlers that will be executed before any service's contract.
func WithGlobalHandlers(handlers ...HandlerFunc) Option {
	return func(s *edgeConfig) {
		s.globalHandlers = handlers
	}
}

func WithTrace(tp Tracer) Option {
	return func(s *edgeConfig) {
		s.tracer = tp
	}
}

// WithConnDelegate registers the delegate to receive callbacks on connection open/close events.
// This delegate could be useful to add metrics based on the connections, or any other advanced
// scenarios. For most use cases, this is not necessary.
func WithConnDelegate(d ConnDelegate) Option {
	return func(s *edgeConfig) {
		s.connDelegate = d
	}
}
