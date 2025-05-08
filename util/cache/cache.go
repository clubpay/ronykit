package cache

import (
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

type Cache struct {
	prefix string
	c      *ristretto.Cache[string, any]
}

func New() (*Cache, error) {
	c, err := ristretto.NewCache[string, any](
		&ristretto.Config[string, any]{
			NumCounters: 100_000,
			MaxCost:     10_000,
			BufferItems: 64,
		},
	)
	if err != nil {
		return nil, err
	}

	return &Cache{c: c}, nil
}

func (c *Cache) Partition(key string) *Cache {
	return &Cache{
		prefix: key + ":",
		c:      c.c,
	}
}

func (c *Cache) Get(key string) (any, bool) {
	return c.c.Get(c.prefix + key)
}

func (c *Cache) Set(key string, val any) bool {
	return c.c.Set(c.prefix+key, val, 1)
}

func (c *Cache) SetTTL(key string, val any, ttl time.Duration) bool {
	return c.c.SetWithTTL(c.prefix+key, val, 1, ttl)
}

func (c *Cache) Purge(key string) {
	c.c.Del(c.prefix + key)
}
