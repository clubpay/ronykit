package rediscluster

import (
	"context"
	"fmt"

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

var _ kit.Cluster = (*cluster)(nil)

func New(opts ...Option) (kit.Cluster, error) {
	c := &cluster{}
	for _, o := range opts {
		o(c)
	}

	return c, nil
}

func MustNew(opts ...Option) kit.Cluster {
	c, err := New(opts...)
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
	c.id = id
	c.ps = c.rc.Subscribe(context.Background(), fmt.Sprintf("%s:chan:%s", c.prefix, id))
	c.d = d
}

func (c *cluster) Publish(id string, data []byte) error {
	return c.rc.Publish(context.Background(), fmt.Sprintf("%s:chan:%s", c.prefix, id), data).Err()
}

func (c *cluster) SetKey(key, value string) error {
	return c.rc.Set(
		context.Background(),
		fmt.Sprintf("%s:kv:%s", c.prefix, key), value,
		redis.KeepTTL,
	).Err()
}

func (c *cluster) DeleteKey(key string) error {
	return c.rc.Del(
		context.Background(),
		fmt.Sprintf("%s:kv:%s", c.prefix, key),
	).Err()
}

func (c *cluster) GetKey(key string) (string, error) {
	return c.rc.Get(
		context.Background(),
		fmt.Sprintf("%s:kv:%s", c.prefix, key),
	).Result()
}
