package rpc

type Option func(b *bundle)

func Listen(protoAddr string) Option {
	return func(b *bundle) {
		b.listen = protoAddr
	}
}

func Decoder(f DecoderFunc) Option {
	return func(b *bundle) {
		b.decoder = f
	}
}
