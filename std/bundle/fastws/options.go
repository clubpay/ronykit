package fastws

import (
	"github.com/clubpay/ronykit"
)

type Option func(b *bundle)

func WithLogger(l ronykit.Logger) Option {
	return func(b *bundle) {
		b.l = l
	}
}

func Listen(protoAddr string) Option {
	return func(b *bundle) {
		b.listen = protoAddr
	}
}

func WithPredicateKey(key string) Option {
	return func(b *bundle) {
		b.predicateKey = key
	}
}

func WithEncoding(enc ronykit.Encoding) Option {
	return func(b *bundle) {
		b.enc = enc
	}
}

func WithCustomRPC(in ronykit.IncomingRPCFactory, out ronykit.OutgoingRPCFactory) Option {
	return func(b *bundle) {
		b.rpcInFactory = in
		b.rpcOutFactory = out
	}
}
