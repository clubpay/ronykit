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

// WithCustomRPC lets you define your RPC message container. RPC message container will be
// represented as Envelope in your application. RPCContainer only defines the behavior of
// serialization and deserialization of the Envelope.
func WithCustomRPC(in ronykit.IncomingRPCFactory, out ronykit.OutgoingRPCFactory) Option {
	return func(b *bundle) {
		b.rpcInFactory = in
		b.rpcOutFactory = out
	}
}
