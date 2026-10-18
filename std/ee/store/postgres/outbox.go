package postgres

import (
	"cmp"
	"context"
	"slices"
	"time"

	"github.com/clubpay/ronykit/ee/store"
	"github.com/clubpay/ronykit/std/ee/store/postgres/internal/db"
	"github.com/clubpay/ronykit/x/rkit"

	"github.com/jackc/pgx/v5"
)

// Claim leases outbox entries on the primary. Claims are serialized by a
// transaction-scoped advisory lock: they are short, and it guarantees that
// the per-key ordering check never races with another claim in flight.
func (s *Store) Claim(ctx context.Context, limit int, leaseFor time.Duration) ([]store.OutboxEntry, error) {
	var rows []db.ClaimOutboxRow

	err := pgx.BeginTxFunc(ctx, s.primary, pgx.TxOptions{}, func(ptx pgx.Tx) error {
		q := db.New(ptx)
		if err := q.LockOutboxClaims(ctx); err != nil {
			return err
		}

		var err error

		rows, err = q.ClaimOutbox(ctx, db.ClaimOutboxParams{
			LeaseSeconds: leaseFor.Seconds(),
			MaxEntries:   int32(min(limit, 1<<31-1)),
		})

		return err
	})
	if err != nil {
		return nil, mapError(err)
	}

	// RETURNING has no order.
	slices.SortFunc(rows, func(a, b db.ClaimOutboxRow) int { return cmp.Compare(a.Seq, b.Seq) })

	out := rkit.Map(rows, func(r db.ClaimOutboxRow) store.OutboxEntry {
		return store.OutboxEntry{
			ID:           r.ID,
			PartitionKey: r.PartitionKey,
			Data:         r.Data,
			CreatedAt:    r.CreatedAt.Time,
		}
	})

	return out, nil
}

func (s *Store) Ack(ctx context.Context, ids ...string) error {
	return mapError(db.New(s.primary).AckOutbox(ctx, ids))
}
