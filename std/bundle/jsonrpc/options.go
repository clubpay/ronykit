package jsonrpc

type Option func(b *bundle)

func Listen(protoAddr string) Option {
	return func(b *bundle) {
		b.listen = protoAddr
	}
}
