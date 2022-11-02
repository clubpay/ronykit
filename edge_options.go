package ronykit

import "time"

type Option func(s *EdgeServer)

func WithLogger(l Logger) Option {
	return func(s *EdgeServer) {
		s.l = l
	}
}

// RegisterGateway lets you register a bundle in constructor of the EdgeServer. However,
// you still can EdgeServer.RegisterGateway method after the constructor. But you must
// be noticed that this is method is not concurrent safe.
func RegisterGateway(gw ...Gateway) Option {
	return func(s *EdgeServer) {
		for _, b := range gw {
			s.RegisterGateway(b)
		}
	}
}

func RegisterCluster(id string, cb ClusterBackend) Option {
	return func(s *EdgeServer) {
		s.RegisterCluster(id, cb)
	}
}

func RegisterService(services ...Service) Option {
	return func(s *EdgeServer) {
		for _, svc := range services {
			s.RegisterService(svc)
		}
	}
}

// WithShutdownTimeout sets the maximum time to wait until all running request to finish.
// Default is to wait forever.
func WithShutdownTimeout(d time.Duration) Option {
	return func(s *EdgeServer) {
		s.shutdownTimeout = d
	}
}

// WithErrorHandler registers a global error handler to catch any error that
// happens before EdgeServer can deliver incoming message to the handler, or delivering
// the outgoing message to the client.
// Internal errors are usually wrapped and could be checked for better error handling.
// You can check with errors.Is function to see if the error is one of the following:
// ErrDispatchFailed, ErrWriteToClosedConn, ErrNoHandler
// ErrDecodeIncomingMessageFailed,, ErrEncodeOutgoingMessageFailed
func WithErrorHandler(h ErrHandler) Option {
	return func(s *EdgeServer) {
		s.eh = h
	}
}

// WithGlobalHandlers sets the handlers that will be executed before any service's contract.
func WithGlobalHandlers(handlers ...HandlerFunc) Option {
	return func(s *EdgeServer) {
		s.gh = handlers
	}
}
