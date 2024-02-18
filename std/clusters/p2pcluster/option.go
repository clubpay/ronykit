package p2pcluster

import (
	"time"

	"github.com/clubpay/ronykit/kit"
)

type Option func(*cluster)

func WithBroadcastInterval(interval time.Duration) Option {
	return func(c *cluster) {
		c.broadcastInterval = interval
	}
}

func WithListenHost(host string) Option {
	return func(c *cluster) {
		c.listenHost = host
	}
}

func WithListenPort(port int) Option {
	return func(c *cluster) {
		c.listenPort = port
	}
}

func WithLogger(log kit.Logger) Option {
	return func(c *cluster) {
		c.log = log
	}
}
