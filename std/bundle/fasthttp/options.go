package fasthttp

type Option func(r *bundle)

func WithServerName(name string) Option {
	return func(r *bundle) {
		r.srv.Name = name
	}
}

func Listen(addr string) Option {
	return func(r *bundle) {
		r.listen = addr
	}
}

func WithCORS(cfg CORSConfig) Option {
	return func(r *bundle) {
		r.cors = newCORS(cfg)
	}
}

func WithPredicateKey(key string) Option {
	return func(b *bundle) {
		b.predicateKey = key
	}
}
