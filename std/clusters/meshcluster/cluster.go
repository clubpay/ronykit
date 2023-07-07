package meshcluster

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/hashicorp/memberlist"
	"github.com/hashicorp/raft"
)

type cluster struct {
	cfg     config
	fsm     fsm
	r       *raft.Raft
	addr    raft.ServerAddress
	members *memberlist.Memberlist
}

var (
	_ kit.Cluster      = (*cluster)(nil)
	_ kit.ClusterStore = (*cluster)(nil)
)

func MustNew(name string, opts ...Option) kit.Cluster {
	c, err := New(name, opts...)
	if err != nil {
		panic(err)
	}

	return c
}

func New(name string, opts ...Option) (kit.Cluster, error) {
	cfg := config{
		name:   name,
		dbPath: filepath.Join(utils.GetExecDir(), "stores"),
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
	raftStoreDbPath := filepath.Join(c.cfg.dbPath, "raft")
	_ = os.MkdirAll(raftStoreDbPath, 0o644)
	raftStore, err := newBadgerRaftStore(raftStoreDbPath)
	if err != nil {
		return err
	}
	kvStoreDbPath := filepath.Join(c.cfg.dbPath, "kv")
	kvStore, err := newBadgerKeyValueStore(kvStoreDbPath)
	if err != nil {
		return err
	}

	ssStoreDbPath := filepath.Join(c.cfg.dbPath, "snapshots")
	_ = os.MkdirAll(ssStoreDbPath, 0o644)
	ssStore, err := raft.NewFileSnapshotStore(ssStoreDbPath, 1, nil)
	if err != nil {
		return err
	}

	trans, err := raft.NewTCPTransport("", nil, 0, 0, nil)
	if err != nil {
		return err
	}

	theFSM := newFSM(kvStore)
	r, err := raft.NewRaft(raftCfg, theFSM, raftStore, raftStore, ssStore, trans)
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

	if err := c.members.Shutdown(); err != nil {
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
	return c
}

func (c *cluster) Set(key, value string, ttl time.Duration) error {
	c.r.VerifyLeader()
}

func (c *cluster) Delete(key string) error {
	// TODO implement me
	panic("implement me")
}

func (c *cluster) Get(key string) (string, error) {
	// TODO implement me
	panic("implement me")
}

func (c *cluster) Scan(prefix string, cb func(string) bool) error {
	// TODO implement me
	panic("implement me")
}

func (c *cluster) ScanWithValue(prefix string, cb func(string, string) bool) error {
	// TODO implement me
	panic("implement me")
}
