package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/std/memories/sqlstore"
	_ "modernc.org/sqlite"
)

// Config configures a SQLite memory backend.
type Config struct {
	DSN      string
	Embedder intent.Embedder
	Migrate  bool
}

// Memory is a SQLite-backed session memory store.
type Memory struct {
	inner *sqlstore.Memory
	db    *sql.DB
}

// Open connects to SQLite and returns a memory store.
// DSN examples: "file:memory.db" or "file::memory:?cache=shared".
func Open(ctx context.Context, cfg Config) (*Memory, error) {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = "file:intent-memory.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite memory: open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("sqlite memory: ping database: %w", err)
	}

	inner, err := sqlstore.New(ctx, db, sqlstore.DialectSQLite, cfg.Embedder, cfg.Migrate)
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
