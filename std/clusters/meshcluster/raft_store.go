package meshcluster

import (
	"encoding/binary"
	"sync"
	"sync/atomic"

	"github.com/cenkalti/backoff/v4"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/dgraph-io/badger/v4"
	"github.com/goccy/go-json"
	"github.com/hashicorp/raft"
)

var (
	firstIndexKey = []byte("IDX_F")
	lastIndexKey  = []byte("IDX_L")
)

type badgerRaftStore struct {
	mtx        utils.SpinLock
	db         *badger.DB
	firstIndex uint64
	lastIndex  uint64
	updateDB   func(fn func(txn *badger.Txn) error) error
	viewDB     func(fn func(txn *badger.Txn) error) error
}

var (
	_ raft.LogStore    = (*badgerRaftStore)(nil)
	_ raft.StableStore = (*badgerRaftStore)(nil)
)

func newBadgerRaftStore(path string) (*badgerRaftStore, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}

	s := &badgerRaftStore{
		db:       db,
		updateDB: updateDB(db),
		viewDB:   viewDB(db),
	}
	s.loadFirstAndLastIndex()

	return s, nil
}

func (store *badgerRaftStore) loadFirstAndLastIndex() {
	err := store.viewDB(
		func(txn *badger.Txn) error {
			// FirstIndex
			firstIndexItem, err := txn.Get(firstIndexKey)
			switch err {
			case nil:
				return firstIndexItem.Value(
					func(val []byte) error {
						store.firstIndex = binary.BigEndian.Uint64(val)

						return nil
					},
				)
			case badger.ErrKeyNotFound:
			default:
				return err
			}

			// LastIndex
			lastIndexItem, err := txn.Get(lastIndexKey)
			switch err {
			case nil:
				return lastIndexItem.Value(
					func(val []byte) error {
						store.lastIndex = binary.BigEndian.Uint64(val)

						return nil
					},
				)
			case badger.ErrKeyNotFound:
			default:
				return err
			}

			return nil
		},
	)
	if err != nil {
		panic("failed to load indices")
	}
}

func (store *badgerRaftStore) FirstIndex() (uint64, error) {
	store.mtx.Lock()
	defer store.mtx.Unlock()

	return store.firstIndex, nil
}

func (store *badgerRaftStore) LastIndex() (uint64, error) {
	store.mtx.Lock()
	defer store.mtx.Unlock()

	return store.lastIndex, nil
}

func (store *badgerRaftStore) GetLog(index uint64, log *raft.Log) error {
	return store.viewDB(
		func(txn *badger.Txn) error {
			key := logKey(index)
			item, err := txn.Get(key[:])
			if err != nil {
				return err
			}

			return item.Value(
				func(val []byte) error {
					return json.Unmarshal(val, log)
				},
			)
		},
	)
}

func (store *badgerRaftStore) StoreLog(log *raft.Log) error {
	return store.updateDB(
		func(txn *badger.Txn) error {
			fi := atomic.LoadUint64(&store.firstIndex)
			li := atomic.LoadUint64(&store.lastIndex)
			key := logKey(log.Index)
			logBytes, err := json.Marshal(log)
			if err != nil {
				return err
			}

			err = txn.Set(key[:], logBytes)
			if err != nil {
				return err
			}

			if fi == 0 || log.Index < fi {
				if !atomic.CompareAndSwapUint64(&store.firstIndex, fi, log.Index) {
					return badger.ErrConflict
				}

				nfi := utils.UInt64ToBigEndian(log.Index)
				err = txn.Set(firstIndexKey, nfi[:])
				if err != nil {
					return err
				}
			}

			if log.Index > li {
				if !atomic.CompareAndSwapUint64(&store.lastIndex, li, log.Index) {
					return badger.ErrConflict
				}

				nli := utils.UInt64ToBigEndian(log.Index)
				err = txn.Set(lastIndexKey, nli[:])
				if err != nil {
					return err
				}
			}

			return nil
		},
	)
}

func (store *badgerRaftStore) StoreLogs(logs []*raft.Log) error {
	return store.updateDB(
		func(txn *badger.Txn) error {
			fi := atomic.LoadUint64(&store.firstIndex)
			li := atomic.LoadUint64(&store.lastIndex)

			maxIndex := li
			minIndex := fi
			for _, log := range logs {
				key := logKey(log.Index)
				logBytes, err := json.Marshal(log)
				if err != nil {
					return err
				}

				err = txn.Set(key[:], logBytes)
				if err != nil {
					return err
				}

				if log.Index > maxIndex {
					maxIndex = log.Index
				}
				if log.Index < minIndex {
					minIndex = log.Index
				}
			}

			if fi == 0 || minIndex < fi {
				if !atomic.CompareAndSwapUint64(&store.firstIndex, fi, minIndex) {
					return badger.ErrConflict
				}

				nfi := utils.UInt64ToBigEndian(minIndex)
				err := txn.Set(firstIndexKey, nfi[:])
				if err != nil {
					return badger.ErrConflict
				}
			}

			if maxIndex > li {
				if !atomic.CompareAndSwapUint64(&store.lastIndex, li, maxIndex) {
					return badger.ErrConflict
				}

				nli := utils.UInt64ToBigEndian(maxIndex)
				err := txn.Set(lastIndexKey, nli[:])
				if err != nil {
					return err
				}
			}

			return nil
		},
	)
}

func (store *badgerRaftStore) DeleteRange(min, max uint64) error {
	return store.updateDB(
		func(txn *badger.Txn) error {
			for i := min; i <= max; i++ {
				key := logKey(i)
				err := txn.Delete(key[:])
				switch err {
				case nil, badger.ErrKeyNotFound:
				default:
					return err
				}
			}

			return nil
		},
	)
}

func (store *badgerRaftStore) Set(key []byte, val []byte) error {
	return store.updateDB(
		func(txn *badger.Txn) error {
			return txn.Set(key, val)
		},
	)
}

func (store *badgerRaftStore) Get(key []byte) ([]byte, error) {
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

func (store *badgerRaftStore) SetUint64(key []byte, val uint64) error {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], val)

	return store.Set(key, b[:])
}

func (store *badgerRaftStore) GetUint64(key []byte) (uint64, error) {
	b, err := store.Get(key)
	if err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(b), nil
}

func logKey(index uint64) [9]byte {
	var key [9]byte
	key[0] = 'L'
	binary.BigEndian.PutUint64(key[1:], index)

	return key
}

var backoffPool = sync.Pool{}

func updateDB(db *badger.DB) func(func(txn *badger.Txn) error) error {
	x, ok := backoffPool.Get().(backoff.BackOff)
	if !ok {
		x = backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5)
	}
	return func(update func(txn *badger.Txn) error) error {
		err := backoff.Retry(
			func() error { return wrapErr(db.Update(update)) },
			x,
		)

		backoffPool.Put(x)

		return err
	}
}

func viewDB(db *badger.DB) func(func(txn *badger.Txn) error) error {
	x, ok := backoffPool.Get().(backoff.BackOff)
	if !ok {
		x = backoff.WithMaxRetries(backoff.NewExponentialBackOff(), 5)
	}
	return func(view func(txn *badger.Txn) error) error {
		err := backoff.Retry(
			func() error { return wrapErr(db.View(view)) },
			x,
		)

		backoffPool.Put(x)

		return err
	}
}

func wrapErr(err error) error {
	switch err {
	default:
		return backoff.Permanent(err)
	case nil:
		return nil
	case badger.ErrConflict:
		return err
	}
}
