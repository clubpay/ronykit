// Package postgres is a Postgres implementation of the ee store contract
// (github.com/clubpay/ronykit/ee/store), built on pgx and sqlc.
//
// States live in one JSONB table, so the ee codec must produce JSON (the
// default ee.JSONCodec does). Writes, actions and the outbox always use the
// primary; read-only View transactions (ee's direct Get/Find) are spread over
// replicas when any are configured.
package postgres

//go:generate sqlc generate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/clubpay/ronykit/ee/store"
	"github.com/clubpay/ronykit/std/ee/store/postgres/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Environment variables used when no pool or DSN option is given.
const (
	EnvPrimaryDSN  = "EE_POSTGRES_PRIMARY_DSN"
	EnvReplicaDSNs = "EE_POSTGRES_REPLICA_DSNS" // comma separated
)

const defaultFetchSize = 100

// Store implements store.Store.
type Store struct {
	primary    *pgxpool.Pool
	replicas   []*pgxpool.Pool
	owned      []*pgxpool.Pool
	next       atomic.Uint64
	isolation  pgx.TxIsoLevel
	fetchSize  int
	finders    map[string]FindFunc
	migrations []migration
}

var (
	_ store.Store  = (*Store)(nil)
	_ store.Outbox = (*Store)(nil)
)

type config struct {
	primary     *pgxpool.Pool
	primaryDSN  string
	replicas    []*pgxpool.Pool
	replicaDSNs []string
	isolation   pgx.TxIsoLevel
	fetchSize   int
	finders     map[string]FindFunc
	migrations  []migration
}

type Option func(*config)

// WithPrimary uses an existing pool for writes. The caller keeps ownership.
func WithPrimary(p *pgxpool.Pool) Option { return func(c *config) { c.primary = p } }

// WithPrimaryDSN opens (and on Close, closes) a pool for writes.
func WithPrimaryDSN(dsn string) Option { return func(c *config) { c.primaryDSN = dsn } }

// WithReplicas adds read replicas used by View. The caller keeps ownership.
func WithReplicas(p ...*pgxpool.Pool) Option {
	return func(c *config) { c.replicas = append(c.replicas, p...) }
}

// WithReplicaDSNs opens (and on Close, closes) a pool per read replica.
func WithReplicaDSNs(dsn ...string) Option {
	return func(c *config) { c.replicaDSNs = append(c.replicaDSNs, dsn...) }
}

// WithIsolation sets the isolation level of write transactions. The default
// is read committed, which is enough for ee's optimistic version checks.
func WithIsolation(l pgx.TxIsoLevel) Option { return func(c *config) { c.isolation = l } }

// WithFetchSize sets how many rows Find fetches from its cursor at a time.
func WithFetchSize(n int) Option { return func(c *config) { c.fetchSize = n } }

func New(opts ...Option) (*Store, error) {
	cfg := config{isolation: pgx.ReadCommitted, fetchSize: defaultFetchSize, finders: map[string]FindFunc{}}
	for _, o := range opts {
		o(&cfg)
	}

	if cfg.primary == nil && cfg.primaryDSN == "" {
		cfg.primaryDSN = os.Getenv(EnvPrimaryDSN)
	}

	if len(cfg.replicas) == 0 && len(cfg.replicaDSNs) == 0 {
		for dsn := range strings.SplitSeq(os.Getenv(EnvReplicaDSNs), ",") {
			if dsn = strings.TrimSpace(dsn); dsn != "" {
				cfg.replicaDSNs = append(cfg.replicaDSNs, dsn)
			}
		}
	}

	s := &Store{
		primary:    cfg.primary,
		replicas:   cfg.replicas,
		isolation:  cfg.isolation,
		fetchSize:  cfg.fetchSize,
		finders:    cfg.finders,
		migrations: cfg.migrations,
	}

	open := func(dsn string) (*pgxpool.Pool, error) {
		p, err := pgxpool.New(context.Background(), dsn)
		if err != nil {
			return nil, err
		}

		s.owned = append(s.owned, p)

		return p, nil
	}

	if s.primary == nil {
		if cfg.primaryDSN == "" {
			return nil, errors.New(
				"postgres: no primary: use WithPrimary, WithPrimaryDSN or " + EnvPrimaryDSN,
			)
		}

		p, err := open(cfg.primaryDSN)
		if err != nil {
			return nil, fmt.Errorf("postgres: primary: %w", err)
		}

		s.primary = p
	}

	for _, dsn := range cfg.replicaDSNs {
		p, err := open(dsn)
		if err != nil {
			_ = s.Close()

			return nil, fmt.Errorf("postgres: replica: %w", err)
		}

		s.replicas = append(s.replicas, p)
	}

	return s, nil
}

func MustNew(opts ...Option) *Store {
	s, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return s
}

// Close closes the pools the store opened from DSNs; pools passed with
// WithPrimary/WithReplicas stay open.
func (s *Store) Close() error {
	for _, p := range s.owned {
		p.Close()
	}

	s.owned = nil

	return nil
}

// Primary is the write pool, e.g. for running sqlc queries outside ee.
func (s *Store) Primary() *pgxpool.Pool { return s.primary }

func (s *Store) Outbox() store.Outbox { return s }

func (s *Store) Tx(ctx context.Context, fn func(ctx context.Context, tx store.Tx) error) error {
	return s.run(ctx, s.primary, pgx.TxOptions{IsoLevel: s.isolation}, false, fn)
}

// View runs fn in a read-only, repeatable-read transaction on a replica (round
// robin), or on the primary if there is none or ctx came from ForcePrimary.
// Replicas may lag behind the primary.
func (s *Store) View(ctx context.Context, fn func(ctx context.Context, tx store.Tx) error) error {
	opts := pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}

	return s.run(ctx, s.readPool(ctx), opts, true, fn)
}

func (s *Store) run(
	ctx context.Context,
	pool *pgxpool.Pool,
	opts pgx.TxOptions,
	readOnly bool,
	fn func(ctx context.Context, tx store.Tx) error,
) error {
	err := pgx.BeginTxFunc(ctx, pool, opts, func(ptx pgx.Tx) error {
		return fn(ctx, &tx{s: s, ptx: ptx, q: db.New(ptx), readOnly: readOnly})
	})

	return mapError(err)
}

func (s *Store) readPool(ctx context.Context) *pgxpool.Pool {
	if len(s.replicas) == 0 || forcedPrimary(ctx) {
		return s.primary
	}

	return s.replicas[s.next.Add(1)%uint64(len(s.replicas))]
}

type forcePrimaryKey struct{}

// ForcePrimary makes View read from the primary, for read-your-writes right
// after a write: orders.Get(postgres.ForcePrimary(ctx), id).
func ForcePrimary(ctx context.Context) context.Context {
	return context.WithValue(ctx, forcePrimaryKey{}, true)
}

func forcedPrimary(ctx context.Context) bool {
	v, _ := ctx.Value(forcePrimaryKey{}).(bool)

	return v
}

// PgxTx unwraps a transaction handed out by this store (e.g. from
// ActionCtx.Tx or ee.Delivery.Tx), so sqlc-generated queries can run in it:
//
//	if ptx, ok := postgres.PgxTx(ctx.Tx()); ok {
//		rows, err := mydb.New(ptx).ListOpenOrders(ctx)
//	}
func PgxTx(t store.Tx) (pgx.Tx, bool) {
	pt, ok := t.(*tx)
	if !ok {
		return nil, false
	}

	return pt.ptx, true
}

// mapError translates Postgres errors ee must react to: serialization
// failures and deadlocks become store.ErrConflict (ee retries the unit of
// work); writes in read-only transactions become store.ErrReadOnly.
func mapError(err error) error {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return err
	}

	switch pgErr.Code {
	case "40001", "40P01": // serialization_failure, deadlock_detected
		return fmt.Errorf("%w: %w", store.ErrConflict, err)
	case "25006": // read_only_sql_transaction
		return fmt.Errorf("%w: %w", store.ErrReadOnly, err)
	default:
		return err
	}
}
