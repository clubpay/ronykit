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

func WithErrorHandler(h ErrHandler) Option {
	return func(s *EdgeServer) {
		s.eh = h
	}
}
