package filelog

import log "github.com/clubpay/ronykit/x/telemetry/logkit"

type Option func(c *core)

func MaxSize(maxSize int) Option {
	return func(c *core) {
		c.maxSize = maxSize
	}
}

func MaxAge(maxAge int) Option {
	return func(c *core) {
		c.maxAge = maxAge
	}
}

func MaxBackups(n int) Option {
	return func(c *core) {
		c.maxBackups = n
	}
}

func CustomEncoder(enc log.Encoder) Option {
	return func(c *core) {
		c.enc = enc
	}
}

func LogLevel(lvl log.Level) Option {
	return func(c *core) {
		c.lvl = lvl
	}
}
