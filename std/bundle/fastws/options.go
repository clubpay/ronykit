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
