package rediscluster

import (
	"context"
	"fmt"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/go-redis/redis/v8"
)

// KeepTTL is used in Set method of the ClusterStore interface.
const KeepTTL = -1

type cluster struct {
	rc           *redis.Client
	ps           *redis.PubSub
	msgChan      <-chan *redis.Message
	id           string
	d            kit.ClusterDelegate
	shutdownChan <-chan struct{}
	prefix       string
	idleTime     time.Duration
	gcPeriod     time.Duration
}

var (
	_ kit.Cluster      = (*cluster)(nil)
	_ kit.ClusterStore = (*cluster)(nil)
)

func New(name string, opts ...Option) (kit.Cluster, error) {
	c := &cluster{
		prefix:   name,
		idleTime: time.Minute * 10,
		gcPeriod: time.Minute,
	}
	for _, o := range opts {
		o(c)
	}

	go c.gc()

	return c, nil
}

func MustNew(name string, opts ...Option) kit.Cluster {
	c, err := New(name, opts...)
	if err != nil {
		panic(err)
	}

	return c
}

func (c *cluster) gc() {
	key := fmt.Sprintf("%s:gc", c.prefix)
	instancesKey := fmt.Sprintf("%s:instances", c.prefix)
	ctx := context.Background()
	idleSec := int64(c.idleTime / time.Second)
	for {
		if c.id != "" {
			c.rc.HSet(ctx, instancesKey, c.id, utils.TimeUnix())
		}

		time.Sleep(c.gcPeriod)

		ok, _ := c.rc.SetNX(context.Background(), key, "running", c.idleTime).Result()
		if ok {
			members, _ := c.rc.HGetAll(ctx, instancesKey).Result()
			now := utils.TimeUnix()
			for k, v := range members {
				if now-utils.StrToInt64(v) > idleSec {
					c.rc.HDel(ctx, instancesKey, k)
				}
			}
		}
	}
}

func (c *cluster) Start(ctx context.Context) error {
	c.msgChan = c.ps.Channel()
	go func() {
		for {
			select {
			case msg := <-c.msgChan:
				_ = c.d.OnMessage(utils.S2B(msg.Payload))
			case <-c.shutdownChan:
				_ = c.ps.Close()

				return
			}
		}
	}()

	return nil
}

func (c *cluster) Shutdown(ctx context.Context) error {
	return c.rc.HDel(ctx, fmt.Sprintf("%s:instances", c.prefix), c.id).Err()
}

func (c *cluster) Subscribe(id string, d kit.ClusterDelegate) {
	ctx := context.Background()
	c.id = id
	c.ps = c.rc.Subscribe(ctx, fmt.Sprintf("%s:chan:%s", c.prefix, id))
	c.rc.HSet(ctx, fmt.Sprintf("%s:instances", c.prefix), id, utils.TimeUnix())
	c.d = d
}

func (c *cluster) Publish(id string, data []byte) error {
	return c.rc.Publish(context.Background(), fmt.Sprintf("%s:chan:%s", c.prefix, id), data).Err()
}

func (c *cluster) Store() kit.ClusterStore {
	return c
}

func (c *cluster) Set(key, value string, ttl time.Duration) error {
	return c.rc.Set(
		context.Background(),
		fmt.Sprintf("%s:kv:%s", c.prefix, key), value,
		ttl,
	).Err()
}

func (c *cluster) Delete(key string) error {
	return c.rc.Del(
		context.Background(),
		fmt.Sprintf("%s:kv:%s", c.prefix, key),
	).Err()
}

func (c *cluster) Get(key string) (string, error) {
	return c.rc.Get(
		context.Background(),
		fmt.Sprintf("%s:kv:%s", c.prefix, key),
	).Result()
}

func (c *cluster) Scan(prefix string, cb func(string) bool) error {
	ctx := context.Background()
	iter := c.rc.Scan(ctx, 0, fmt.Sprintf("%s*", prefix), 512).Iterator()

	for iter.Next(ctx) {
		if !cb(iter.Val()) {
			return nil
		}
	}

	return nil
}

func (c *cluster) Subscribers() ([]string, error) {
	ctx := context.Background()

	return c.rc.HKeys(ctx, fmt.Sprintf("%s:instances", c.prefix)).Result()
}
