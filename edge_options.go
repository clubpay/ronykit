package ronykit

type Option func(s *EdgeServer)

func WithLogger(l Logger) Option {
	return func(s *EdgeServer) {
		s.l = l
	}
}

// RegisterBundle lets you register a bundle in constructor of the EdgeServer. However,
// you still can EdgeServer.RegisterBundle method after the constructor. But you must
// be noticed that this is method is not concurrent safe.
func RegisterBundle(bundles ...Bundle) Option {
	return func(s *EdgeServer) {
		for _, b := range bundles {
			s.RegisterBundle(b)
		}
	}
}

func RegisterService(services ...Service) Option {
	return func(s *EdgeServer) {
		for _, svc := range services {
			s.RegisterService(svc)
		}
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
