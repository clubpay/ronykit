package rediscluster

import (
	"context"
	"fmt"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/redis/go-redis/v9"
)

// KeepTTL is used in Set method of the ClusterStore interface.
const KeepTTL = -1

type cluster struct {
	rc         *redis.Client
	ps         *redis.PubSub
	msgChan    <-chan *redis.Message
	id         string
	d          kit.ClusterDelegate
	shutdownFn context.CancelFunc
	prefix     string
	idleTime   time.Duration
	gcPeriod   time.Duration
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

	return c, nil
}

func MustNew(name string, opts ...Option) kit.Cluster {
	c, err := New(name, opts...)
	if err != nil {
		panic(err)
	}

	return c
}

func (c *cluster) gc(ctx context.Context) {
	key := fmt.Sprintf("%s:gc", c.prefix)
	instancesKey := fmt.Sprintf("%s:instances", c.prefix)
	idleSec := int64(c.idleTime / time.Second)

	_ = luaGC.Run(
		ctx, c.rc,
		[]string{
			key,
			instancesKey,
		},
		c.id,
		idleSec,
		utils.TimeUnix(),
	).Err()
}

func (c *cluster) Start(ctx context.Context) error {
	runCtx, cf := context.WithCancel(context.Background())
	c.shutdownFn = cf

	c.ps = c.rc.Subscribe(ctx, fmt.Sprintf("%s:chan:%s", c.prefix, c.id))
	c.rc.HSet(ctx, fmt.Sprintf("%s:instances", c.prefix), c.id, utils.TimeUnix())
	c.msgChan = c.ps.Channel()
	gcTimer := time.NewTimer(c.gcPeriod)
	go func() {
		for {
			select {
			case <-gcTimer.C:
				c.gc(runCtx)
			case msg, ok := <-c.msgChan:
				if !ok {
					return
				}
				go c.d.OnMessage(utils.S2B(msg.Payload))
			case <-runCtx.Done():
				_ = c.ps.Close()

				return
			}
		}
	}()

	return nil
}

func (c *cluster) Shutdown(ctx context.Context) error {
	c.shutdownFn()

	return c.rc.HDel(ctx, fmt.Sprintf("%s:instances", c.prefix), c.id).Err()
}

func (c *cluster) Subscribe(id string, d kit.ClusterDelegate) {
	c.id = id
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
	iter := c.rc.Scan(ctx, 0, fmt.Sprintf("%s:kv:%s*", c.prefix, prefix), 512).Iterator()

	for iter.Next(ctx) {
		if !cb(iter.Val()) {
			return nil
		}
	}

	return nil
}

func (c *cluster) ScanWithValue(prefix string, cb func(string, string) bool) error {
	ctx := context.Background()
	iter := c.rc.Scan(ctx, 0, fmt.Sprintf("%s:kv:%s*", c.prefix, prefix), 512).Iterator()

	for iter.Next(ctx) {
		v, err := c.rc.Get(ctx, iter.Val()).Result()
		if err != nil {
			return err
		}
		if !cb(iter.Val(), v) {
			return nil
		}
	}

	return nil
}

func (c *cluster) Subscribers() ([]string, error) {
	ctx := context.Background()

	return c.rc.HKeys(ctx, fmt.Sprintf("%s:instances", c.prefix)).Result()
}
