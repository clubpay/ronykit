package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/clubpay/ronykit/std/ee/store/postgres/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "embed"
)

//go:embed sql/schema.sql
var schemaSQL string

type migration struct {
	name string
	fn   func(ctx context.Context, tx pgx.Tx) error
}

// WithMigration registers an application migration, e.g. indexes for the
// queries a service runs:
//
//	postgres.WithMigration("001-order-indexes", func(ctx context.Context, tx pgx.Tx) error {
//		_, err := tx.Exec(ctx, postgres.FieldIndex("order", "merchant", postgres.Text))
//		return err
//	})
//
// Migrate runs each name once, in registration order, after the ee schema.
func WithMigration(name string, fn func(ctx context.Context, tx pgx.Tx) error) Option {
	return func(c *config) { c.migrations = append(c.migrations, migration{name: name, fn: fn}) }
}

// Migrate creates the ee tables if needed and applies pending WithMigration
// steps, in one transaction on the primary. Concurrent replicas booting at
// once are serialized by an advisory lock.
func (s *Store) Migrate(ctx context.Context) error {
	return pgx.BeginTxFunc(ctx, s.primary, pgx.TxOptions{}, func(ptx pgx.Tx) error {
		q := db.New(ptx)
		if err := q.LockMigrations(ctx); err != nil {
			return err
		}

		if _, err := ptx.Exec(ctx, schemaSQL); err != nil {
			return fmt.Errorf("postgres: ee schema: %w", err)
		}

		for _, m := range s.migrations {
			done, err := q.MigrationApplied(ctx, m.name)
			if err != nil {
				return err
			}

			if done {
				continue
			}

			if err := m.fn(ctx, ptx); err != nil {
				return fmt.Errorf("postgres: migration %q: %w", m.name, err)
			}

			if err := q.RecordMigration(ctx, m.name); err != nil {
				return err
			}
		}

		return nil
	})
}

// Check fails fast at boot: every pool must be reachable and see the ee
// tables, and every registered migration must be applied.
func (s *Store) Check(ctx context.Context) error {
	pools := append([]*pgxpool.Pool{s.primary}, s.replicas...)

	var errs []error

	for i, p := range pools {
		role := "primary"
		if i > 0 {
			role = fmt.Sprintf("replica %d", i)
		}

		var ok bool

		err := p.QueryRow(ctx,
			`SELECT to_regclass('ee_states') IS NOT NULL AND to_regclass('ee_outbox') IS NOT NULL`,
		).Scan(&ok)

		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("postgres: %s: %w", role, err))
		case !ok:
			errs = append(errs, fmt.Errorf("postgres: %s: ee tables missing, run Migrate", role))
		}
	}

	q := db.New(s.primary)
	for _, m := range s.migrations {
		done, err := q.MigrationApplied(ctx, m.name)

		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("postgres: migration %q: %w", m.name, err))
		case !done:
			errs = append(errs, fmt.Errorf("postgres: migration %q not applied, run Migrate", m.name))
		}
	}

	return errors.Join(errs...)
}
