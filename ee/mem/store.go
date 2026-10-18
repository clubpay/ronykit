// Package mem provides in-memory implementations of the ee bus and store
// contracts, for tests and small single-process deployments.
package mem

import (
	"context"
	"iter"
	"slices"
	"sync"
	"time"

	"github.com/clubpay/ronykit/ee/store"
	"github.com/clubpay/ronykit/x/rkit"
)

// Store is an in-memory store.Store. Transactions are optimistic: writes are
// buffered and validated against the committed versions at commit, so
// concurrent writers get store.ErrConflict exactly like with a real backend.
type Store struct {
	mu     sync.RWMutex
	data   map[key]store.Record
	outbox []*outboxItem
	decode Decoder
	now    func() time.Time
}

type key struct{ kind, id string }

type outboxItem struct {
	entry       store.OutboxEntry
	leasedUntil time.Time
}

type StoreOption func(*Store)

// WithDecoder sets how Find reads fields out of encoded records. It must
// match the ee codec; the default decodes JSON.
func WithDecoder(d Decoder) StoreOption { return func(s *Store) { s.decode = d } }

// WithStoreClock overrides time.Now for outbox leases.
func WithStoreClock(now func() time.Time) StoreOption { return func(s *Store) { s.now = now } }

func NewStore(opts ...StoreOption) *Store {
	s := &Store{data: map[key]store.Record{}, decode: JSONDecoder, now: time.Now}
	for _, o := range opts {
		o(s)
	}

	return s
}

var (
	_ store.Store  = (*Store)(nil)
	_ store.Outbox = (*Store)(nil)
)

func (s *Store) Tx(ctx context.Context, fn func(ctx context.Context, tx store.Tx) error) error {
	t := &tx{s: s, writes: map[key]*write{}, checks: map[key]uint64{}}
	if err := fn(ctx, t); err != nil {
		return err
	}

	return s.commit(t)
}

func (s *Store) View(ctx context.Context, fn func(ctx context.Context, tx store.Tx) error) error {
	return fn(ctx, &tx{s: s, readOnly: true})
}

func (s *Store) Outbox() store.Outbox { return s }

func (s *Store) Close() error { return nil }

// Len returns the number of committed records of kind.
func (s *Store) Len(kind string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n := 0

	for k := range s.data {
		if k.kind == kind {
			n++
		}
	}

	return n
}

func (s *Store) commit(t *tx) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, want := range t.checks {
		if s.data[k].Version != want {
			return store.ErrConflict
		}
	}

	for k, w := range t.writes {
		if w.deleted {
			delete(s.data, k)
		} else {
			s.data[k] = w.rec
		}
	}

	for _, e := range t.outbox {
		s.outbox = append(s.outbox, &outboxItem{entry: e})
	}

	return nil
}

func (s *Store) Claim(_ context.Context, limit int, leaseFor time.Duration) ([]store.OutboxEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	// blocked holds keys with an older entry this call cannot return, so
	// their newer entries must wait.
	blocked := map[string]bool{}

	var out []store.OutboxEntry

	for _, it := range s.outbox {
		if len(out) == limit {
			break
		}

		k := it.entry.PartitionKey
		if it.leasedUntil.After(now) || blocked[k] {
			if k != "" {
				blocked[k] = true
			}

			continue
		}

		it.leasedUntil = now.Add(leaseFor)
		out = append(out, it.entry)
	}

	return out, nil
}

func (s *Store) Ack(_ context.Context, ids ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.outbox = slices.DeleteFunc(s.outbox, func(it *outboxItem) bool {
		return rkit.Contains(ids, it.entry.ID)
	})

	return nil
}

type write struct {
	rec     store.Record
	deleted bool
}

type tx struct {
	s        *Store
	readOnly bool
	writes   map[key]*write
	// checks holds, per written key, the committed version this transaction
	// observed (0 = absent). Commit fails if any of them moved.
	checks map[key]uint64
	outbox []store.OutboxEntry
}

func (t *tx) committed(k key) (store.Record, bool) {
	t.s.mu.RLock()
	defer t.s.mu.RUnlock()

	r, ok := t.s.data[k]

	return r, ok
}

func (t *tx) lookup(k key) (store.Record, bool) {
	if w, ok := t.writes[k]; ok {
		return w.rec, !w.deleted
	}

	return t.committed(k)
}

// observe records, the first time k is written, the committed version the
// write was based on (0 = absent). It must be the version the caller just
// looked up, not a fresh read: another commit may have landed in between.
func (t *tx) observe(k key, version uint64) {
	if _, ok := t.checks[k]; !ok {
		t.checks[k] = version
	}
}

func (t *tx) Get(_ context.Context, kind, id string) (store.Record, error) {
	r, ok := t.lookup(key{kind, id})
	if !ok {
		return store.Record{}, store.ErrNotFound
	}

	return r, nil
}

func (t *tx) Insert(_ context.Context, rec store.Record) error {
	if t.readOnly {
		return store.ErrReadOnly
	}

	k := key{rec.Kind, rec.ID}
	if _, ok := t.lookup(k); ok {
		return store.ErrConflict
	}

	t.observe(k, 0)

	rec.Version = 1
	t.writes[k] = &write{rec: rec}

	return nil
}

func (t *tx) Update(_ context.Context, rec store.Record) error {
	if t.readOnly {
		return store.ErrReadOnly
	}

	k := key{rec.Kind, rec.ID}

	cur, ok := t.lookup(k)
	if !ok {
		return store.ErrNotFound
	}

	if cur.Version != rec.Version {
		return store.ErrConflict
	}

	t.observe(k, cur.Version)

	rec.Version++
	t.writes[k] = &write{rec: rec}

	return nil
}

func (t *tx) Delete(_ context.Context, kind, id string, version uint64) error {
	if t.readOnly {
		return store.ErrReadOnly
	}

	k := key{kind, id}

	cur, ok := t.lookup(k)
	if !ok {
		return store.ErrNotFound
	}

	if cur.Version != version {
		return store.ErrConflict
	}

	t.observe(k, cur.Version)
	t.writes[k] = &write{rec: cur, deleted: true}

	return nil
}

func (t *tx) AppendOutbox(_ context.Context, entries ...store.OutboxEntry) error {
	if t.readOnly {
		return store.ErrReadOnly
	}

	t.outbox = append(t.outbox, entries...)

	return nil
}

// Find scans every record of the kind; it is meant for tests and small data.
func (t *tx) Find(_ context.Context, q store.Query) iter.Seq2[store.Record, error] {
	return func(yield func(store.Record, error) bool) {
		recs, err := t.find(q)
		if err != nil {
			yield(store.Record{}, err)

			return
		}

		for _, r := range recs {
			if !yield(r, nil) {
				return
			}
		}
	}
}

func (t *tx) find(q store.Query) ([]store.Record, error) {
	merged := map[key]store.Record{}

	t.s.mu.RLock()

	for k, r := range t.s.data {
		if k.kind == q.Kind {
			merged[k] = r
		}
	}

	t.s.mu.RUnlock()

	for k, w := range t.writes {
		switch {
		case k.kind != q.Kind:
		case w.deleted:
			delete(merged, k)
		default:
			merged[k] = w.rec
		}
	}

	type row struct {
		rec  store.Record
		doc  map[string]any
		sort string // id, for a stable default order
	}

	rows := make([]row, 0, len(merged))

	for _, r := range merged {
		doc, err := t.s.decode(r.Data)
		if err != nil {
			return nil, err
		}

		if matchAll(doc, q.Where) {
			rows = append(rows, row{rec: r, doc: doc, sort: r.ID})
		}
	}

	slices.SortFunc(rows, func(a, b row) int {
		for _, o := range q.OrderBy {
			c := compareDoc(a.doc[o.Field], b.doc[o.Field])
			if o.Desc {
				c = -c
			}

			if c != 0 {
				return c
			}
		}

		return compareDoc(a.sort, b.sort)
	})

	if q.Limit > 0 && len(rows) > q.Limit {
		rows = rows[:q.Limit]
	}

	out := rkit.Map(rows, func(r row) store.Record { return r.rec })

	return out, nil
}
