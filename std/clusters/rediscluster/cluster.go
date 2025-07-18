package rediscluster

import (
	"context"
	"fmt"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/redis/go-redis/v9"
)

// KeepTTL can be used in `Set` method of the ClusterStore interface.
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
	const (
		defaultIdleTime = time.Minute * 10
		defaultGCPeriod = time.Minute
	)

	c := &cluster{
		prefix:   name,
		idleTime: defaultIdleTime,
		gcPeriod: defaultGCPeriod,
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

	_ = luaGC.Run( //nolint:errcheck
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

func (c *cluster) Start(ctx context.Context) error { //nolint:contextcheck
	c.ps = c.rc.Subscribe(ctx, fmt.Sprintf("%s:chan:%s", c.prefix, c.id))
	c.rc.HSet(ctx, fmt.Sprintf("%s:instances", c.prefix), c.id, utils.TimeUnix())
	c.msgChan = c.ps.Channel()
	gcTimer := time.NewTimer(c.gcPeriod)

	runCtx, cf := context.WithCancel(context.Background())
	c.shutdownFn = cf

	go func() {
		for {
			select {
			case <-gcTimer.C:
				c.gc(runCtx)
			case msg, ok := <-c.msgChan:
				if !ok {
					return
				}

				c.d.OnMessage(utils.S2B(msg.Payload))
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

func (c *cluster) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.rc.Set(
		ctx,
		fmt.Sprintf("%s:kv:%s", c.prefix, key), value,
		ttl,
	).Err()
}

func (c *cluster) SetMulti(ctx context.Context, m map[string]string, ttl time.Duration) error {
	pipe := c.rc.Pipeline()
	for k, v := range m {
		pipe.Set(
			ctx,
			fmt.Sprintf("%s:kv:%s", c.prefix, k), v,
			ttl,
		)
	}

	_, err := pipe.Exec(ctx)

	return err
}

func (c *cluster) Delete(ctx context.Context, key string) error {
	return c.rc.Del(
		ctx,
		fmt.Sprintf("%s:kv:%s", c.prefix, key),
	).Err()
}

func (c *cluster) Get(ctx context.Context, key string) (string, error) {
	return c.rc.Get(
		ctx,
		fmt.Sprintf("%s:kv:%s", c.prefix, key),
	).Result()
}

func (c *cluster) Scan(ctx context.Context, prefix string, cb func(string) bool) error {
	const scanSize = 512

	iter := c.rc.Scan(ctx, 0, fmt.Sprintf("%s:kv:%s*", c.prefix, prefix), scanSize).Iterator()

	for iter.Next(ctx) {
		if !cb(iter.Val()) {
			return nil
		}
	}

	return nil
}

func (c *cluster) ScanWithValue(ctx context.Context, prefix string, cb func(string, string) bool) error {
	const scanSize = 512

	iter := c.rc.Scan(ctx, 0, fmt.Sprintf("%s:kv:%s*", c.prefix, prefix), scanSize).Iterator()

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
