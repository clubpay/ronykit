package meshcluster

import (
	"io"

	"github.com/hashicorp/raft"
)

type fsm struct{}

var _ raft.FSM = (*fsm)(nil)

func (f fsm) Apply(log *raft.Log) interface{} {
	//TODO implement me
	panic("implement me")
}

func (f fsm) Snapshot() (raft.FSMSnapshot, error) {
	//TODO implement me
	panic("implement me")
}

func (f fsm) Restore(snapshot io.ReadCloser) error {
	//TODO implement me
	panic("implement me")
}
