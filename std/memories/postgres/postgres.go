package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/std/memories/sqlstore"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config configures a Postgres memory backend.
type Config struct {
	DSN      string
	Embedder intent.Embedder
	Migrate  bool
}

// Memory is a Postgres-backed session memory store.
type Memory struct {
	inner *sqlstore.Memory
	db    *sql.DB
}

// Open connects to Postgres and returns a memory store.
func Open(ctx context.Context, cfg Config) (*Memory, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("postgres memory: DSN is required")
	}

	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("postgres memory: open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("postgres memory: ping database: %w", err)
	}

	inner, err := sqlstore.New(ctx, db, sqlstore.DialectPostgres, cfg.Embedder, cfg.Migrate)
	if err != nil {
		_ = db.Close()

		return nil, err
	}

	return &Memory{inner: inner, db: db}, nil
}

func (m *Memory) ForSession(sessionID string) intent.SessionMemory {
	return m.inner.ForSession(sessionID)
}

// Close closes the underlying database connection.
func (m *Memory) Close() error {
	if m == nil || m.db == nil {
		return nil
	}

	return m.db.Close()
}

var _ intent.Memory = (*Memory)(nil)
