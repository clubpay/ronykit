package meshcluster

import (
	"time"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/dgraph-io/badger/v4"
)

type badgerKVStore struct {
	mtx        utils.SpinLock
	db         *badger.DB
	firstIndex uint64
	lastIndex  uint64
	updateDB   func(fn func(txn *badger.Txn) error) error
	viewDB     func(fn func(txn *badger.Txn) error) error
}

func newBadgerKeyValueStore(path string) (*badgerKVStore, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}

	s := &badgerKVStore{
		db:       db,
		updateDB: updateDB(db),
		viewDB:   viewDB(db),
	}

	return s, nil
}

func (store *badgerKVStore) Set(key []byte, val []byte) error {
	return store.updateDB(
		func(txn *badger.Txn) error {
			return txn.Set(key, val)
		},
	)
}

func (store *badgerKVStore) SetTTL(key []byte, val []byte, ttl time.Duration) error {
	return store.updateDB(
		func(txn *badger.Txn) error {
			return txn.SetEntry(badger.NewEntry(key, val).WithTTL(ttl))
		},
	)
}

func (store *badgerKVStore) Get(key []byte) ([]byte, error) {
	var val []byte
	err := store.viewDB(
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

func (store *badgerKVStore) Del(key []byte) error {
	return store.updateDB(
		func(txn *badger.Txn) error {
			return txn.Delete(key)
		},
	)
}
