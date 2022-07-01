package stub

import "time"

type Option func(cfg *config)

type config struct {
	hostPort      string
	secure        bool
	skipVerifyTLS bool

	readTimeout, writeTimeout time.Duration
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
