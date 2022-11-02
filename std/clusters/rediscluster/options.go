package rediscluster

import "github.com/go-redis/redis/v8"

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
