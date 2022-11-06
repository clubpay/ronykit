package rediscluster

import (
	"context"
	"fmt"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/go-redis/redis/v8"
)

type cluster struct {
	rc           *redis.Client
	ps           *redis.PubSub
	msgChan      <-chan *redis.Message
	id           string
	d            kit.ClusterDelegate
	shutdownChan <-chan struct{}
	prefix       string
}

var (
	_ kit.Cluster      = (*cluster)(nil)
	_ kit.ClusterStore = (*cluster)(nil)
)

func New(name string, opts ...Option) (kit.Cluster, error) {
	c := &cluster{
		prefix: name,
	}
	for _, o := range opts {
		o(c)
	}

	return c, nil
}

func MustNew(name string, opts ...Option) kit.Cluster {
	c, err := New(name, opts...)
	if err != nil {
		panic(err)
	}

	return c
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
	return nil
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

func (c *cluster) Subscribers() ([]string, error) {
	ctx := context.Background()
	return c.rc.HKeys(ctx, fmt.Sprintf("%s:instances", c.prefix)).Result()
}
