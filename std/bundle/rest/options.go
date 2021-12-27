package rest

import "github.com/valyala/fasthttp"

type Option func(r *bundle)

func WithHttpServer(srv *fasthttp.Server) Option {
	return func(r *bundle) {
		r.srv = srv
	}
}

func Listen(addr string) Option {
	return func(r *bundle) {
		r.listen = addr
	}
}
