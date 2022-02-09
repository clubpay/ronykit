package rpc

import (
	log "github.com/ronaksoft/golog"
)

type Option func(b *bundle)

func Logger(l log.Logger) Option {
	return func(b *bundle) {
		b.l = l
	}
}
func Listen(protoAddr string) Option {
	return func(b *bundle) {
		b.listen = protoAddr
	}
}

func PredicateKey(key string) Option {
	return func(b *bundle) {
		b.predicateKey = key
	}
}
