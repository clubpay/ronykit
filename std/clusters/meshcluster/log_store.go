package meshcluster

import (
	"encoding/binary"

	"github.com/dgraph-io/badger/v3"
	"github.com/hashicorp/raft"
)

type logStore struct {
	db *badger.DB
}

var _ raft.LogStore = (*logStore)(nil)

func newLogStore(path string) (*logStore, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}

	return &logStore{
		db: db,
	}, nil
}

func key(index uint64) []byte {
	var key [9]byte
	key[0] = 'L'
	binary.BigEndian.PutUint64(key[1:], index)

	return key[:]
}

func (l logStore) FirstIndex() (uint64, error) {
	err := l.db.View(
		func(txn *badger.Txn) error {
			item, err := txn.Get(key(0))
			if err != nil {
				return err
			}

			return item.Value(func(val []byte) error {

			})
		},
	)
	if err != nil {
		return 0, err
	}
}

func (l logStore) LastIndex() (uint64, error) {
	//TODO implement me
	panic("implement me")
}

func (l logStore) GetLog(index uint64, log *raft.Log) error {
	//TODO implement me
	panic("implement me")
}

func (l logStore) StoreLog(log *raft.Log) error {
	//TODO implement me
	panic("implement me")
}

func (l logStore) StoreLogs(logs []*raft.Log) error {
	//TODO implement me
	panic("implement me")
}

func (l logStore) DeleteRange(min, max uint64) error {
	//TODO implement me
	panic("implement me")
}
