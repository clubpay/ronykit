package rest

import "github.com/valyala/fasthttp"

type Option func(r *rest)

func WithHttpServer(srv *fasthttp.Server) Option {
	return func(r *rest) {
		r.srv = srv
	}
}

func Listen(addr string) Option {
	return func(r *rest) {
		r.listen = addr
	}
}
