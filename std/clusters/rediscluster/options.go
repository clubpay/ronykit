package rediscluster

import (
	"time"

	"github.com/clubpay/ronykit/x/rkit"

	"github.com/redis/go-redis/v9"
)

type Option func(c *cluster)

func WithPrefix(prefix string) Option {
	return func(c *cluster) {
		c.prefix = prefix
	}
}

func WithRedisClient(rc *redis.Client) Option {
	return func(c *cluster) {
		c.rc = rc
	}
}

func WithRedisURL(url string) Option {
	return func(c *cluster) {
		opt := rkit.Must(redis.ParseURL(url))

		c.rc = redis.NewClient(opt)
	}
}

func WithGCPeriod(d time.Duration) Option {
	return func(c *cluster) {
		c.gcPeriod = d
	}
}
