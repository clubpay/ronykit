package meshcluster

import "github.com/hashicorp/raft"

type stableStore struct{}

var _ raft.StableStore = (*stableStore)(nil)

func (s stableStore) Set(key []byte, val []byte) error {
	//TODO implement me
	panic("implement me")
}

func (s stableStore) Get(key []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (s stableStore) SetUint64(key []byte, val uint64) error {
	//TODO implement me
	panic("implement me")
}

func (s stableStore) GetUint64(key []byte) (uint64, error) {
	//TODO implement me
	panic("implement me")
}
