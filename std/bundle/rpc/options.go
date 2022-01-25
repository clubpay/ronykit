package rpc

type Option func(b *bundle)

func Listen(protoAddr string) Option {
	return func(b *bundle) {
		b.listen = protoAddr
	}
}

func Decoder(f func(data []byte, e *Envelope) error) Option {
	return func(b *bundle) {
		b.decoder = f
	}
}
