package meshcluster

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/hashicorp/raft"
)

type cluster struct {
	fsm  raft.FSM
	r    *raft.Raft
	addr raft.ServerAddress
}

func MustNew(name string, opts ...Option) kit.Cluster {
	c, err := New(name, opts...)
	if err != nil {
		panic(err)
	}

	return c
}

func New(name string, opts ...Option) (kit.Cluster, error) {
	raftCfg := raft.DefaultConfig()
	logStore := &logStore{}
	stableStore := &stableStore{}
	ssStore := raft.NewInmemSnapshotStore()

	addr, trans := raft.NewTCPTransport()
	fsm := &fsm{}

	r, err := raft.NewRaft(raftCfg, fsm, logStore, stableStore, ssStore, trans)
	if err != nil {
		return nil, err
	}

	c := &cluster{
		r:    r,
		addr: addr,
	}

	return c, nil
}
