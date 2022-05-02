package storebased

import (
	"context"
	"encoding"
	"net"
	"time"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/utils"
	"github.com/goccy/go-json"
)

const (
	idEmpty     = ""
	storageKey  = "r-kit"
	pathMembers = "all-m"
	pathActives = "all-act"
)

type config struct {
	id                string
	urls              []string
	endpoint          string
	heartbeatInterval time.Duration
	store             ronykit.ClusterStore
}

type member struct {
	ID       string   `json:"id"`
	URLs     []string `json:"urls"`
	Endpoint string   `json:"endpoint"`
}

var (
	_ ronykit.ClusterMember    = (*member)(nil)
	_ encoding.BinaryMarshaler = (*member)(nil)
)

func (m member) ServerID() string {
	return m.ID
}

func (m member) AdvertisedURL() []string {
	return m.URLs
}

func (m member) Dial(ctx context.Context) (net.Conn, error) {
	panic("implement me")
}

func (m *member) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, m)
}

func (m member) MarshalBinary() (data []byte, err error) {
	return json.Marshal(m)
}

type cluster struct {
	cfg          config
	shutdownChan chan struct{}
	d            ronykit.ClusterDelegate
	s            ronykit.ClusterStore
	me           member
}

var _ ronykit.Cluster = (*cluster)(nil)

func New(opts ...Option) (ronykit.Cluster, error) {
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	c := &cluster{
		cfg: cfg,
		me: member{
			ID:       cfg.id,
			URLs:     cfg.urls,
			Endpoint: cfg.endpoint,
		},
		s:            cfg.store,
		shutdownChan: make(chan struct{}),
	}

	return c, nil
}

func (c cluster) Start(ctx context.Context) error {
	x, ok := c.s.(interface {
		Start(ctx context.Context) error
	})
	if ok {
		err := x.Start(ctx)
		if err != nil {
			return err
		}
	}

	go c.heartbeat()

	return nil
}

func (c cluster) heartbeat() {
	timer := time.NewTimer(c.cfg.heartbeatInterval)

	err := c.s.SetMember(context.Background(), c.me)
	if err != nil {
		c.d.OnError(err)
	}

	for {
		err = c.s.SetLastActive(context.Background(), c.me.ID, utils.TimeUnix())
		if err != nil {
			c.d.OnError(err)
		}

		select {
		case <-c.shutdownChan:
			return
		case <-timer.C:
		}
	}
}

func (c cluster) Shutdown(ctx context.Context) error {
	x, ok := c.s.(interface {
		Shutdown(ctx context.Context) error
	})
	if ok {
		return x.Shutdown(ctx)
	}

	return nil
}

func (c cluster) Members(ctx context.Context) ([]ronykit.ClusterMember, error) {
	return c.s.GetActiveMembers(ctx, utils.TimeUnixSubtract(utils.TimeUnix(), c.cfg.heartbeatInterval*3))
}

func (c cluster) MemberByID(ctx context.Context, id string) (ronykit.ClusterMember, error) {
	return c.s.GetMember(ctx, id)
}

func (c cluster) Me() ronykit.ClusterMember {
	return c.me
}

func (c *cluster) Subscribe(d ronykit.ClusterDelegate) {
	c.d = d
}
