package cluster

import (
	"context"
	"time"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/utils"
)

const (
	defaultHeartbeat = time.Second * 30
	defaultTTL       = time.Minute
)

type config struct {
	id       string
	urls     []string
	endpoint string
	hb       time.Duration
	ttl      time.Duration
	store    ronykit.ClusterStore
}

type bundle struct {
	shutdownChan chan struct{}
	d            ronykit.ClusterDelegate
	s            ronykit.ClusterStore
	me           member
	hb           time.Duration
	ttl          time.Duration
}

var _ ronykit.Cluster = (*bundle)(nil)

func New(opts ...Option) (ronykit.Cluster, error) {
	cfg := config{
		id:       utils.RandomID(24),
		urls:     nil,
		endpoint: "",
		hb:       defaultHeartbeat,
		ttl:      defaultTTL,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	c := &bundle{
		me: member{
			ID:       cfg.id,
			URLs:     cfg.urls,
			Endpoint: cfg.endpoint,
		},
		s:            cfg.store,
		hb:           cfg.hb,
		ttl:          cfg.ttl,
		shutdownChan: make(chan struct{}),
	}

	return c, nil
}

func MustNew(opts ...Option) ronykit.Cluster {
	c, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return c
}

func (b *bundle) Start(ctx context.Context) error {
	x, ok := b.s.(interface {
		Start(ctx context.Context) error
	})
	if ok {
		err := x.Start(ctx)
		if err != nil {
			return err
		}
	}

	go b.heartbeat()

	return nil
}

func (b bundle) heartbeat() {
	timer := time.NewTimer(b.hb)

	err := b.s.SetMember(context.Background(), b.me)
	if err != nil {
		b.d.OnError(err)
	}

	for {
		err = b.s.SetLastActive(context.Background(), b.me.ID, utils.TimeUnix())
		if err != nil {
			b.d.OnError(err)
		}

		select {
		case <-b.shutdownChan:
			return
		case <-timer.C:
		}
	}
}

func (b *bundle) Shutdown(ctx context.Context) error {
	x, ok := b.s.(interface {
		Shutdown(ctx context.Context) error
	})
	if ok {
		return x.Shutdown(ctx)
	}

	return nil
}

func (b *bundle) Members(ctx context.Context) ([]ronykit.ClusterMember, error) {
	return b.s.GetActiveMembers(ctx, utils.TimeUnixSubtract(utils.TimeUnix(), b.ttl))
}

func (b *bundle) MemberByID(ctx context.Context, id string) (ronykit.ClusterMember, error) {
	return b.s.GetMember(ctx, id)
}

func (b *bundle) Me() ronykit.ClusterMember {
	return b.me
}

func (b *bundle) Subscribe(d ronykit.ClusterDelegate) {
	b.d = d
}

func (b *bundle) Register(
	serviceName, contractID string, enc ronykit.Encoding, sel ronykit.RouteSelector, input ronykit.Message,
) {
	return
}

func (b *bundle) Dispatch(ctx *ronykit.Context, in []byte) (ronykit.ExecuteArg, error) {
	return ronykit.NoExecuteArg, nil
}
