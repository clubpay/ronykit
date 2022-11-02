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

func New(opts ...Option) (kit.ClusterBackend, error) {
	c := &cluster{}
	for _, o := range opts {
		o(c)
	}

	return c, nil
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
}

func (c *cluster) Publish(id string, data []byte) error {
	return c.rc.Publish(context.Background(), fmt.Sprintf("%s:chan:%s", c.prefix, id), data).Err()
}

var _ kit.ClusterBackend = (*cluster)(nil)
