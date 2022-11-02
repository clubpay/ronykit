package stub

import (
	"io"
	"time"

	"github.com/clubpay/ronykit/kit"
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

	readTimeout, writeTimeout, dialTimeout time.Duration
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
