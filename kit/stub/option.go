package stub

import (
	"io"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpproxy"
	"golang.org/x/net/http/httpproxy"
)

type Option func(cfg *config)

type config struct {
	name          string
	hostPort      string
	secure        bool
	skipVerifyTLS bool
	dumpReq       io.Writer
	dumpRes       io.Writer
	l             kit.Logger
	tp            kit.TracePropagator

	readTimeout, writeTimeout, dialTimeout time.Duration
	proxy                                  *httpproxy.Config
	dialFunc                               fasthttp.DialFunc
}

func Secure() Option {
	return func(cfg *config) {
		cfg.secure = true
	}
}

func SkipTLSVerify() Option {
	return func(s *config) {
		s.skipVerifyTLS = true
	}
}

func Name(name string) Option {
	return func(cfg *config) {
		cfg.name = name
	}
}

func WithReadTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.readTimeout = timeout
	}
}

func WithWriteTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.writeTimeout = timeout
	}
}

func WithDialTimeout(timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.dialTimeout = timeout
	}
}

func WithLogger(l kit.Logger) Option {
	return func(cfg *config) {
		cfg.l = l
	}
}

func DumpTo(w io.Writer) Option {
	return func(cfg *config) {
		cfg.dumpReq = w
		cfg.dumpRes = w
	}
}

func WithTracePropagator(tp kit.TracePropagator) Option {
	return func(cfg *config) {
		cfg.tp = tp
	}
}

// WithHTTPProxy returns an Option that sets the dialer to the provided HTTP proxy.
// example formats:
//
//	localhost:9050
//	username:password@localhost:9050
//	localhost:9050
func WithHTTPProxy(proxyURL string, timeout time.Duration) Option {
	return func(cfg *config) {
		cfg.proxy = httpproxy.FromEnvironment()
		cfg.proxy.HTTPProxy = proxyURL
		cfg.proxy.HTTPSProxy = proxyURL
		cfg.dialFunc = fasthttpproxy.FasthttpHTTPDialerTimeout(proxyURL, timeout)
	}
}

// WithSocksProxy returns an Option that sets the dialer to the provided SOCKS5 proxy.
// example format: localhost:9050
func WithSocksProxy(proxyURL string) Option {
	return func(cfg *config) {
		cfg.proxy = httpproxy.FromEnvironment()
		cfg.proxy.HTTPProxy = proxyURL
		cfg.proxy.HTTPSProxy = proxyURL
		cfg.dialFunc = fasthttpproxy.FasthttpSocksDialer(proxyURL)
	}
}
