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

type logStore struct {
	mtx        utils.SpinLock
	db         *badger.DB
	firstIndex uint64
	lastIndex  uint64
	updateDB   func(fn func(txn *badger.Txn) error) error
	viewDB     func(fn func(txn *badger.Txn) error) error
}

var _ raft.LogStore = (*logStore)(nil)

func newLogStore(path string) (*logStore, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, err
	}

	s := &logStore{
		db:       db,
		updateDB: updateDB(db),
		viewDB:   viewDB(db),
	}
	s.loadFirstAndLastIndex()

	return s, nil
}

func (l *logStore) loadFirstAndLastIndex() {
	err := l.viewDB(
		func(txn *badger.Txn) error {
			// FirstIndex
			firstIndexItem, err := txn.Get(firstIndexKey)
			switch err {
			case nil:
				return firstIndexItem.Value(
					func(val []byte) error {
						l.firstIndex = binary.BigEndian.Uint64(val)

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
						l.lastIndex = binary.BigEndian.Uint64(val)

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

func (l *logStore) FirstIndex() (uint64, error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	return l.firstIndex, nil
}

func (l *logStore) LastIndex() (uint64, error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	return l.lastIndex, nil
}

func (l *logStore) GetLog(index uint64, log *raft.Log) error {
	return l.viewDB(
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

func (l *logStore) StoreLog(log *raft.Log) error {
	return l.updateDB(
		func(txn *badger.Txn) error {
			fi := atomic.LoadUint64(&l.firstIndex)
			li := atomic.LoadUint64(&l.lastIndex)
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
				if !atomic.CompareAndSwapUint64(&l.firstIndex, fi, log.Index) {
					return badger.ErrConflict
				}

				nfi := utils.UInt64ToBigEndian(log.Index)
				err = txn.Set(firstIndexKey, nfi[:])
				if err != nil {
					return err
				}
			}

			if log.Index > li {
				if !atomic.CompareAndSwapUint64(&l.lastIndex, li, log.Index) {
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

func (l *logStore) StoreLogs(logs []*raft.Log) error {
	return l.updateDB(
		func(txn *badger.Txn) error {
			fi := atomic.LoadUint64(&l.firstIndex)
			li := atomic.LoadUint64(&l.lastIndex)

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
				if !atomic.CompareAndSwapUint64(&l.firstIndex, fi, minIndex) {
					return badger.ErrConflict
				}

				nfi := utils.UInt64ToBigEndian(minIndex)
				err := txn.Set(firstIndexKey, nfi[:])
				if err != nil {
					return badger.ErrConflict
				}
			}

			if maxIndex > li {
				if !atomic.CompareAndSwapUint64(&l.lastIndex, li, maxIndex) {
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

func (l *logStore) DeleteRange(min, max uint64) error {
	return l.updateDB(
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
