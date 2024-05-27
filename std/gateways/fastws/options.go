package fastws

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/gobwas/ws"
)

type Option func(b *bundle)

func WithLogger(l kit.Logger) Option {
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
func WithCustomRPC(in kit.IncomingRPCFactory, out kit.OutgoingRPCFactory) Option {
	return func(b *bundle) {
		b.rpcInFactory = in
		b.rpcOutFactory = out
	}
}

// WithWebsocketTextMode sets the write mode of the websocket to text opCode
// This is the default behavior
func WithWebsocketTextMode() Option {
	return func(b *bundle) {
		b.writeMode = ws.OpText
	}
}

// WithWebsocketBinaryMode sets the write mode of the websocket to binary opCode
func WithWebsocketBinaryMode() Option {
	return func(b *bundle) {
		b.writeMode = ws.OpBinary
	}
}
