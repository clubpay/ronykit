package meshcluster

import (
	"github.com/hashicorp/raft"
	"github.com/weaveworks/mesh"
)

type store struct {
	raft.StableStore
}

func (s store) Encode() [][]byte {
	//TODO implement me
	panic("implement me")
}

func (s store) Merge(data mesh.GossipData) mesh.GossipData {
	//TODO implement me
	panic("implement me")
}

var _ mesh.GossipData = (*store)(nil)
