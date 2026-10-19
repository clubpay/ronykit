package redis

import (
	"context"
	"iter"
	"slices"
	"strconv"

	"github.com/clubpay/ronykit/ee/store"

	goredis "github.com/redis/go-redis/v9"
)

type entityKey struct{ kind, id string }

type write struct {
	rec     store.Record
	deleted bool
}

type tx struct {
	s        *Store
	c        goredis.Cmdable
	readOnly bool
	writes   map[entityKey]*write
	// checks holds, per written key, the committed version this transaction
	// observed (0 = absent). Commit fails if any of them moved.
	checks map[entityKey]uint64
	outbox []store.OutboxEntry
}

func newTx(s *Store, c goredis.Cmdable, readOnly bool) *tx {
	return &tx{
		s:        s,
		c:        c,
		readOnly: readOnly,
		writes:   map[entityKey]*write{},
		checks:   map[entityKey]uint64{},
	}
}

func (t *tx) lookup(ctx context.Context, k entityKey) (store.Record, bool, error) {
	if w, ok := t.writes[k]; ok {
		return w.rec, !w.deleted, nil
	}

	return t.s.load(ctx, t.c, k.kind, k.id)
}

// observe records, the first time k is written, the committed version the
// write was based on (0 = absent).
func (t *tx) observe(k entityKey, version uint64) {
	if _, ok := t.checks[k]; !ok {
		t.checks[k] = version
	}
}

func (t *tx) Get(ctx context.Context, kind, id string) (store.Record, error) {
	rec, ok, err := t.lookup(ctx, entityKey{kind, id})
	if err != nil {
		return store.Record{}, err
	}

	if !ok {
		return store.Record{}, store.ErrNotFound
	}

	return rec, nil
}

func (t *tx) Insert(ctx context.Context, rec store.Record) error {
	if t.readOnly {
		return store.ErrReadOnly
	}

	k := entityKey{rec.Kind, rec.ID}
	if _, ok, err := t.lookup(ctx, k); err != nil {
		return err
	} else if ok {
		return store.ErrConflict
	}

	t.observe(k, 0)

	rec.Version = 1
	t.writes[k] = &write{rec: rec}

	return nil
}

func (t *tx) Update(ctx context.Context, rec store.Record) error {
	if t.readOnly {
		return store.ErrReadOnly
	}

	k := entityKey{rec.Kind, rec.ID}

	cur, ok, err := t.lookup(ctx, k)
	if err != nil {
		return err
	}

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

func (t *tx) Delete(ctx context.Context, kind, id string, version uint64) error {
	if t.readOnly {
		return store.ErrReadOnly
	}

	k := entityKey{kind, id}

	cur, ok, err := t.lookup(ctx, k)
	if err != nil {
		return err
	}

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

func (t *tx) Find(ctx context.Context, q store.Query) iter.Seq2[store.Record, error] {
	return func(yield func(store.Record, error) bool) {
		recs, err := t.find(ctx, q)
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

func (t *tx) find(ctx context.Context, q store.Query) ([]store.Record, error) {
	ids, err := t.c.SMembers(ctx, t.s.kindKey(q.Kind)).Result()
	if err != nil {
		return nil, err
	}

	merged := map[string]store.Record{}

	for _, id := range ids {
		rec, ok, err := t.s.load(ctx, t.c, q.Kind, id)
		if err != nil {
			return nil, err
		}

		if ok {
			merged[id] = rec
		}
	}

	for k, w := range t.writes {
		if k.kind != q.Kind {
			continue
		}

		if w.deleted {
			delete(merged, k.id)
		} else {
			merged[k.id] = w.rec
		}
	}

	type row struct {
		rec  store.Record
		doc  map[string]any
		sort string
	}

	rows := make([]row, 0, len(merged))

	for _, r := range merged {
		doc, err := store.DecodeJSON(r.Data)
		if err != nil {
			return nil, err
		}

		if store.MatchAll(doc, q.Where) {
			rows = append(rows, row{rec: r, doc: doc, sort: r.ID})
		}
	}

	slices.SortFunc(rows, func(a, b row) int {
		for _, o := range q.OrderBy {
			c := store.CompareValues(a.doc[o.Field], b.doc[o.Field])
			if o.Desc {
				c = -c
			}

			if c != 0 {
				return c
			}
		}

		return store.CompareValues(a.sort, b.sort)
	})

	if q.Limit > 0 && len(rows) > q.Limit {
		rows = rows[:q.Limit]
	}

	out := make([]store.Record, len(rows))
	for i, r := range rows {
		out[i] = r.rec
	}

	return out, nil
}

func (t *tx) commit(ctx context.Context) error {
	if len(t.checks) == 0 && len(t.outbox) == 0 {
		return nil
	}

	args := make([]any, 0, 1+len(t.checks)*2+1+len(t.writes)*6+1+len(t.outbox)*5)
	args = append(args, len(t.checks))

	for k, ver := range t.checks {
		args = append(args, t.s.stateKey(k.kind, k.id), ver)
	}

	args = append(args, len(t.writes))

	for k, w := range t.writes {
		deleted := "0"
		if w.deleted {
			deleted = "1"
		}

		args = append(args,
			t.s.stateKey(k.kind, k.id),
			t.s.kindKey(k.kind),
			k.id,
			deleted,
			strconv.FormatUint(w.rec.Version, 10),
			string(w.rec.Data),
		)
	}

	args = append(args, len(t.outbox))

	for _, e := range t.outbox {
		args = append(args,
			t.s.itemKey(e.ID),
			e.ID,
			e.PartitionKey,
			string(e.Data),
			strconv.FormatInt(e.CreatedAt.UnixNano(), 10),
		)
	}

	res, err := goredis.NewScript(commitLua).Run(
		ctx,
		t.s.primary,
		[]string{t.s.seqKey(), t.s.pendingKey()},
		args...,
	).Text()
	if err != nil {
		return err
	}

	if res == "conflict" {
		return store.ErrConflict
	}

	return nil
}
