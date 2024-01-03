package fasthttp

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp/proxy"
)

type Option func(b *bundle)

func SuperFast() Option {
	return func(b *bundle) {
		b.utils = speedUtil()
	}
}

func WithServerName(name string) Option {
	return func(b *bundle) {
		b.srv.Name = name
	}
}

func WithBufferSize(read, write int) Option {
	return func(b *bundle) {
		b.srv.ReadBufferSize = read
		b.srv.WriteBufferSize = write
	}
}

func WithLogger(l kit.Logger) Option {
	return func(b *bundle) {
		b.l = l
	}
}

func Listen(addr string) Option {
	return func(b *bundle) {
		b.listen = addr
	}
}

func WithCORS(cfg CORSConfig) Option {
	return func(b *bundle) {
		b.cors = newCORS(cfg)
		b.wsUpgrade.CheckOrigin = b.cors.handleWS
	}
}

func WithPredicateKey(key string) Option {
	return func(b *bundle) {
		b.predicateKey = key
	}
}

func WithWebsocketEndpoint(endpoint string) Option {
	return func(b *bundle) {
		b.wsEndpoint = endpoint
	}
}

func WithReverseProxy(path string, opt ...proxy.Option) Option {
	return func(b *bundle) {
		var err error
		b.reverseProxy, err = proxy.NewReverseProxyWith(opt...)
		if err != nil {
			panic(err)
		}
		if path == "" {
			path = "/"
		}

		b.reverseProxyPath = path
	}
}

func WithCustomRPC(in kit.IncomingRPCFactory, out kit.OutgoingRPCFactory) Option {
	return func(b *bundle) {
		b.rpcInFactory = in
		b.rpcOutFactory = out
	}
}

// CompressionLevel is numeric representation of compression level
type CompressionLevel int

// Represents compression level that will be used in the middleware
const (
	CompressionLevelDisabled        CompressionLevel = -1
	CompressionLevelDefault         CompressionLevel = 0
	CompressionLevelBestSpeed       CompressionLevel = 1
	CompressionLevelBestCompression CompressionLevel = 2
)

func WithCompressionLevel(level CompressionLevel) Option {
	return func(b *bundle) {
		b.compress = level
	}
}
