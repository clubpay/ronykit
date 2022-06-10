package fasthttp

import "github.com/clubpay/ronykit"

type Option func(b *bundle)

func WithServerName(name string) Option {
	return func(b *bundle) {
		b.srv.Name = name
	}
}

func WithBufferSize(read, write int) Option {
	return func(b *bundle) {
		b.srv.ReadBufferSize = read
		b.srv.WriteBufferSize = write
	}
}

func WithLogger(l ronykit.Logger) Option {
	return func(b *bundle) {
		b.l = l
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
		b.wsUpgrade.CheckOrigin = b.cors.handleWS
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

func WithCustomRPC(in ronykit.IncomingRPCFactory, out ronykit.OutgoingRPCFactory) Option {
	return func(b *bundle) {
		b.rpcInFactory = in
		b.rpcOutFactory = out
	}
}

func WithReflectTag(tag string) Option {
	return func(b *bundle) {
		SetTag(tag)
	}
}
