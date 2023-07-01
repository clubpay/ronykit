package meshcluster

import (
	"encoding/binary"

	"github.com/dgraph-io/badger/v4"
	"github.com/hashicorp/raft"
)

type badgerStableStore struct {
	db       *badger.DB
	updateDB func(fn func(txn *badger.Txn) error) error
	viewDB   func(fn func(txn *badger.Txn) error) error
}

var _ raft.StableStore = (*badgerStableStore)(nil)

func newStableStore(path string) (*badgerStableStore, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}

	s := &badgerStableStore{
		db:       db,
		updateDB: updateDB(db),
		viewDB:   viewDB(db),
	}

	return s, nil
}

func (s badgerStableStore) Set(key []byte, val []byte) error {
	return s.updateDB(
		func(txn *badger.Txn) error {
			return txn.Set(key, val)
		},
	)
}

func (s badgerStableStore) Get(key []byte) ([]byte, error) {
	var val []byte
	err := s.viewDB(
		func(txn *badger.Txn) error {
			itm, err := txn.Get(key)
			if err != nil {
				return err
			}
			val, err = itm.ValueCopy(nil)

			return err
		},
	)
	if err != nil {
		return nil, err
	}

	return val, nil
}

func (s badgerStableStore) SetUint64(key []byte, val uint64) error {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], val)

	return s.Set(key, b[:])
}

func (s badgerStableStore) GetUint64(key []byte) (uint64, error) {
	b, err := s.Get(key)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(b), nil
}
