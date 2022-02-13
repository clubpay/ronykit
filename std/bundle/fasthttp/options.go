package fasthttp

type Option func(b *bundle)

func WithServerName(name string) Option {
	return func(b *bundle) {
		b.srv.Name = name
	}
}

func Listen(addr string) Option {
	return func(b *bundle) {
		b.listen = addr
	}
}

func WithCORS(cfg CORSConfig) Option {
	return func(b *bundle) {
		b.cors = newCORS(cfg)
	}
}

func WithPredicateKey(key string) Option {
	return func(b *bundle) {
		b.predicateKey = key
	}
}

func WithWebsocketEndpoint(endpoint string) Option {
	return func(b *bundle) {
		b.wsEndpoint = endpoint
	}
}
