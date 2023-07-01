package meshcluster

import (
	"context"

	"github.com/clubpay/ronykit/kit"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
)

type cluster struct {
	cfg     config
	fsm     raft.FSM
	r       *raft.Raft
	addr    raft.ServerAddress
	members *memberlist.Memberlist
}

var _ kit.Cluster = (*cluster)(nil)

func MustNew(name string, opts ...Option) kit.Cluster {
	c, err := New(name, opts...)
	if err != nil {
		panic(err)
	}

	return c
}

func New(name string, opts ...Option) (kit.Cluster, error) {
	cfg := config{
		name: name,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	c := &cluster{
		cfg: cfg,
	}

	if err := c.initGossip(); err != nil {
		return nil, err
	}

	if err := c.initRaft(); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *cluster) initGossip() error {
	var err error
	c.members, err = memberlist.Create(memberlist.DefaultLANConfig())
	if err != nil {
		return err
	}

	joined, err := c.members.Join(c.cfg.gossipSeeds)
	if err != nil {
		return err
	}

	// TODO:: log joined nodes
	_ = joined

	return nil
}

func (c *cluster) initRaft() error {
	raftCfg := raft.DefaultConfig()
	logStore := &logStore{}
	stableStore := &stableStore{}
	ssStore := raft.NewInmemSnapshotStore()

	trans, err := raft.NewTCPTransport("", nil, 0, 0, nil)
	if err != nil {
		return err
	}

	fsm := &fsm{}
	r, err := raft.NewRaft(raftCfg, fsm, logStore, stableStore, ssStore, trans)
	if err != nil {
		return err
	}

	c.r = r

	return nil
}

func (c *cluster) Start(ctx context.Context) error {
	return nil
}

func (c *cluster) Shutdown(ctx context.Context) error {
	shutdownF := c.r.Shutdown()

	err := c.members.Shutdown()
	if err != nil {
		return err
	}

	return shutdownF.Error()
}

func (c *cluster) Subscribe(id string, d kit.ClusterDelegate) {
	// TODO implement me
	panic("implement me")
}

func (c *cluster) Publish(id string, data []byte) error {
	// TODO implement me
	panic("implement me")
}

func (c *cluster) Subscribers() ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (c *cluster) Store() kit.ClusterStore {
	// TODO implement me
	panic("implement me")
}
