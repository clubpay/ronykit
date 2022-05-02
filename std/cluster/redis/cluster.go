package rediscluster

import (
	"context"
	"time"

	"github.com/clubpay/ronykit"
	"github.com/go-redis/redis/v8"
)

const (
	storageKey = "redis-c"
	keyMembers = "mem"
)

type config struct {
	heartbeatInterval time.Duration
	redisDNS          string
}

type cluster struct {
	cfg          config
	redisC       *redis.Client
	shutdownChan chan struct{}
	d            ronykit.ClusterDelegate
}

var _ ronykit.Cluster = (*cluster)(nil)

func New(opts ...Option) (ronykit.Cluster, error) {
	c := &cluster{
		shutdownChan: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(&c.cfg)
	}

	redisDSN, err := redis.ParseURL(c.cfg.redisDNS)
	if err != nil {
		return nil, err
	}
	c.redisC = redis.NewClient(redisDSN)

	return c, nil
}

func (c cluster) Start() error {
	err := c.redisC.Ping(context.Background()).Err()
	if err != nil {
		return err
	}

	go c.heartbeat()

	return nil
}

func (c cluster) heartbeat() {
	timer := time.NewTimer(c.cfg.heartbeatInterval)
	for {
		select {
		case <-c.shutdownChan:
			return
		case <-timer.C:
		}

		key := "" // active status key
		_ = c.redisC.Set(context.Background(), key, true, c.cfg.heartbeatInterval*3).Err()
	}
}

func (c cluster) Shutdown() error {
	return c.redisC.Shutdown(context.Background()).Err()
}

func (c cluster) Members() []ronykit.ClusterMember {
	//TODO implement me
	panic("implement me")
}

func (c cluster) MemberByID(s string) ronykit.ClusterMember {
	//TODO implement me
	panic("implement me")
}

func (c cluster) Me() ronykit.ClusterMember {
	//TODO implement me
	panic("implement me")
}

func (c *cluster) Subscribe(d ronykit.ClusterDelegate) {
	c.d = d
}
