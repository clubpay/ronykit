// Package store defines the persistence contract that ee storage backends
// implement (in-memory in ee/mem, Postgres and Redis under std/ee/store).
//
// Like package bus, it is untyped: backends store Records and never see Go
// entity types. Bootstrapping, migrations and query optimization (indexes,
// Redis secondary sets, Lua scripts, ...) are backend specific and exposed by
// each backend's own API, not by this contract.
package store

import (
	"context"
	"errors"
	"iter"
	"time"

	"github.com/clubpay/ronykit/x/rkit"
)

var (
	ErrNotFound = errors.New("store: not found")
	// ErrConflict means an optimistic version check failed or an inserted
	// record already exists. ee retries the whole unit of work on it.
	ErrConflict = errors.New("store: conflict")
	// ErrReadOnly is returned by writes attempted inside View.
	ErrReadOnly = errors.New("store: read-only transaction")
)

// Record is one persisted entity instance.
type Record struct {
	// Kind is the entity name (EntityName()).
	Kind string
	ID   string
	// Version starts at 1 on insert and is incremented by the store on every
	// successful update.
	Version uint64
	// Data is the entity encoded by the ee codec.
	Data []byte
}

// Store is the contract every storage backend implements.
type Store interface {
	// Tx runs fn in a transaction. Any error returned by fn rolls it back.
	Tx(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
	// View runs fn in a read-only transaction (writes fail with
	// ErrReadOnly). Backends may serve it from a replica, so it can lag
	// behind committed Tx writes.
	View(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
	Outbox() Outbox
	Close() error
}

// Tx is the set of operations available inside a transaction. Writes are
// visible to later reads of the same transaction.
type Tx interface {
	Get(ctx context.Context, kind, id string) (Record, error)
	// Insert stores a new record with Version 1. ErrConflict if it exists.
	Insert(ctx context.Context, rec Record) error
	// Update replaces the record if its stored version equals rec.Version,
	// then bumps the stored version. ErrConflict otherwise.
	Update(ctx context.Context, rec Record) error
	// Delete removes the record if its stored version equals version.
	Delete(ctx context.Context, kind, id string, version uint64) error
	// Find streams the records matching q. The sequence must be consumed
	// before the transaction ends.
	Find(ctx context.Context, q Query) iter.Seq2[Record, error]
	// AppendOutbox stores entries atomically with the rest of the transaction.
	AppendOutbox(ctx context.Context, entries ...OutboxEntry) error
}

// OutboxEntry is an encoded event waiting to be published. The store does not
// interpret Data; ee encodes/decodes the bus envelope itself.
type OutboxEntry struct {
	ID string
	// PartitionKey is the envelope's partition key; Claim uses it to keep
	// per-key publish order.
	PartitionKey string
	Data         []byte
	CreatedAt    time.Time
}

// Outbox is used by the ee relay to move committed entries to the bus.
type Outbox interface {
	// Claim leases up to limit unpublished entries for leaseFor, oldest first.
	// Entries whose lease expires without Ack become claimable again.
	//
	// Claim must preserve per-key order across concurrent relays: it never
	// returns an entry while an older unacknowledged entry with the same
	// non-empty PartitionKey is leased or not returned by the same call.
	// Entries with an empty key are unordered.
	Claim(ctx context.Context, limit int, leaseFor time.Duration) ([]OutboxEntry, error)
	Ack(ctx context.Context, ids ...string) error
}

// OutboxStats is the unpublished outbox backlog, including entries currently
// leased by a relay.
type OutboxStats struct {
	Pending int
	// Oldest is the CreatedAt of the oldest pending entry. Zero when Pending
	// is 0.
	Oldest time.Time
}

// StatsReporter is an optional store capability. Backends that can report
// outbox backlog implement it; ee.Env.Stats uses it when present.
type StatsReporter interface {
	OutboxStats(ctx context.Context) (OutboxStats, error)
}

type primaryReadKey struct{}

// WithPrimary marks ctx so a replicated store serves View from the primary.
// In-memory stores ignore it. Use it for a read that must observe a write
// just committed on this process.
func WithPrimary(ctx context.Context) context.Context {
	return context.WithValue(ctx, primaryReadKey{}, true)
}

// PrimaryRead reports whether ctx asks for a primary read.
func PrimaryRead(ctx context.Context) bool {
	return rkit.TryCast[bool](ctx.Value(primaryReadKey{}))
}
