package rediscluster

import "github.com/redis/go-redis/v9"

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
		opt, err := redis.ParseURL(url)
		if err != nil {
			panic(err)
		}

		c.rc = redis.NewClient(opt)
	}
}
