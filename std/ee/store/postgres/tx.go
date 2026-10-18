package postgres

import (
	"context"
	"errors"
	"iter"

	"github.com/clubpay/ronykit/ee/store"
	"github.com/clubpay/ronykit/std/ee/store/postgres/internal/db"
	"github.com/clubpay/ronykit/x/rkit"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// tx implements store.Tx. Writes are written so that an expected failure
// (duplicate insert, stale version) never raises a Postgres error, which
// would abort the transaction: ee.Dedup and retries rely on the transaction
// staying usable.
type tx struct {
	s        *Store
	ptx      pgx.Tx
	q        *db.Queries
	readOnly bool
	cursors  int
}

func (t *tx) Get(ctx context.Context, kind, id string) (store.Record, error) {
	row, err := t.q.GetState(ctx, db.GetStateParams{Kind: kind, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Record{}, store.ErrNotFound
	}

	if err != nil {
		return store.Record{}, mapError(err)
	}

	return store.Record{Kind: row.Kind, ID: row.ID, Version: uint64(row.Version), Data: row.Data}, nil
}

func (t *tx) Insert(ctx context.Context, rec store.Record) error {
	if t.readOnly {
		return store.ErrReadOnly
	}

	n, err := t.q.InsertState(ctx, db.InsertStateParams{Kind: rec.Kind, ID: rec.ID, Data: rec.Data})
	if err != nil {
		return mapError(err)
	}

	if n == 0 {
		return store.ErrConflict
	}

	return nil
}

func (t *tx) Update(ctx context.Context, rec store.Record) error {
	if t.readOnly {
		return store.ErrReadOnly
	}

	n, err := t.q.UpdateState(ctx, db.UpdateStateParams{
		Kind:    rec.Kind,
		ID:      rec.ID,
		Version: int64(rec.Version),
		Data:    rec.Data,
	})
	if err != nil {
		return mapError(err)
	}

	return t.missOrConflict(ctx, n, rec.Kind, rec.ID)
}

func (t *tx) Delete(ctx context.Context, kind, id string, version uint64) error {
	if t.readOnly {
		return store.ErrReadOnly
	}

	n, err := t.q.DeleteState(ctx, db.DeleteStateParams{Kind: kind, ID: id, Version: int64(version)})
	if err != nil {
		return mapError(err)
	}

	return t.missOrConflict(ctx, n, kind, id)
}

// missOrConflict explains a version-checked write that matched n rows.
func (t *tx) missOrConflict(ctx context.Context, n int64, kind, id string) error {
	if n > 0 {
		return nil
	}

	exists, err := t.q.StateExists(ctx, db.StateExistsParams{Kind: kind, ID: id})
	switch {
	case err != nil:
		return mapError(err)
	case exists:
		return store.ErrConflict
	default:
		return store.ErrNotFound
	}
}

func (t *tx) AppendOutbox(ctx context.Context, entries ...store.OutboxEntry) error {
	if t.readOnly {
		return store.ErrReadOnly
	}

	rows := rkit.Map(entries, func(e store.OutboxEntry) db.AppendOutboxParams {
		return db.AppendOutboxParams{
			ID:           e.ID,
			PartitionKey: e.PartitionKey,
			Data:         e.Data,
			CreatedAt:    pgtype.Timestamptz{Time: e.CreatedAt, Valid: true},
		}
	})

	_, err := t.q.AppendOutbox(ctx, rows)

	return mapError(err)
}

func (t *tx) Find(ctx context.Context, q store.Query) iter.Seq2[store.Record, error] {
	if fn, ok := t.s.finders[q.Kind]; ok {
		return fn(ctx, t.ptx, q)
	}

	return t.find(ctx, q)
}
