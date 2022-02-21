package fasthttp

import "github.com/clubpay/ronykit"

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

func WithEncoding(enc ronykit.Encoding) Option {
	return func(b *bundle) {
		b.enc = enc
	}
}

func WithCustomRPC(in IncomingRPCFactory, out OutgoingRPCFactory) Option {
	// FixME: provide predefined factories
	return func(b *bundle) {
		b.rpcInFactory = in
		b.rpcOutFactory = out
	}
}
