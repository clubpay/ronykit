package rest

type Option func(r *rest)

func WithDecoder(f DecoderFunc) Option {
	return func(r *rest) {
		r.decoder = f
	}
}
