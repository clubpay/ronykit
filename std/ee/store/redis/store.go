// Package redis is a Redis implementation of the ee store contract
// (github.com/clubpay/ronykit/ee/store).
//
// Entities are JSON hashes, one key per record, so the ee codec must produce
// JSON (the default ee.JSONCodec does). Find scans the kind's id set and
// filters in process; it is for modest kinds. A hot kind can replace Find
// with WithFind.
//
// The key prefix contains a Redis hash tag (default "{ee}") so every key of
// one store lands in one cluster slot. Commit, claim and ack are Lua scripts
// and must run on that slot. Replicas of one service share the prefix.
//
// Find lists the kind's id set and filters in memory. That suits modest
// kinds; a large kind wants a purpose-built index outside this store.
package redis

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/clubpay/ronykit/ee/store"
	"github.com/clubpay/ronykit/x/rkit"

	goredis "github.com/redis/go-redis/v9"
)

// Environment variables used when no client or address option is given.
const (
	EnvURL         = "EE_REDIS_URL"
	EnvAddr        = "EE_REDIS_ADDR"
	EnvReplicaURL  = "EE_REDIS_REPLICA_URL"
	EnvReplicaAddr = "EE_REDIS_REPLICA_ADDR"
)

const defaultPrefix = "{ee}"

// Store implements store.Store.
type Store struct {
	primary goredis.Cmdable
	replica goredis.Cmdable
	owned   []*goredis.Client
	prefix  string
}

var (
	_ store.Store         = (*Store)(nil)
	_ store.Outbox        = (*Store)(nil)
	_ store.StatsReporter = (*Store)(nil)
)

type config struct {
	client      goredis.Cmdable
	replica     goredis.Cmdable
	url         string
	addr        string
	replicaURL  string
	replicaAddr string
	prefix      string
}

type Option func(*config)

// WithClient uses an existing client for writes. The caller keeps ownership.
func WithClient(c goredis.Cmdable) Option { return func(cfg *config) { cfg.client = c } }

// WithURL opens (and on Close, closes) a client from a redis URL.
func WithURL(url string) Option { return func(cfg *config) { cfg.url = url } }

// WithAddr opens (and on Close, closes) a client for addr (host:port).
func WithAddr(addr string) Option { return func(cfg *config) { cfg.addr = addr } }

// WithReplicaClient serves View from c unless the context asks for the primary.
func WithReplicaClient(c goredis.Cmdable) Option { return func(cfg *config) { cfg.replica = c } }

// WithReplicaURL opens a read replica from a redis URL.
func WithReplicaURL(url string) Option { return func(cfg *config) { cfg.replicaURL = url } }

// WithReplicaAddr opens a read replica at addr.
func WithReplicaAddr(addr string) Option { return func(cfg *config) { cfg.replicaAddr = addr } }

// WithPrefix isolates this store's keys. It should include a hash tag,
// for example "{billing}". Default "{ee}".
func WithPrefix(p string) Option { return func(cfg *config) { cfg.prefix = p } }

func New(opts ...Option) (*Store, error) {
	cfg := config{prefix: defaultPrefix}
	for _, o := range opts {
		o(&cfg)
	}

	if cfg.client == nil && cfg.url == "" && cfg.addr == "" {
		cfg.url = os.Getenv(EnvURL)
		if cfg.url == "" {
			cfg.addr = os.Getenv(EnvAddr)
		}
	}

	if cfg.replica == nil && cfg.replicaURL == "" && cfg.replicaAddr == "" {
		cfg.replicaURL = os.Getenv(EnvReplicaURL)
		if cfg.replicaURL == "" {
			cfg.replicaAddr = os.Getenv(EnvReplicaAddr)
		}
	}

	if cfg.prefix == "" {
		return nil, errors.New("redis: prefix must not be empty")
	}

	s := &Store{primary: cfg.client, replica: cfg.replica, prefix: cfg.prefix}

	open := func(url, addr string) (goredis.Cmdable, error) {
		var (
			c   *goredis.Client
			err error
		)

		switch {
		case url != "":
			opt, perr := goredis.ParseURL(url)
			if perr != nil {
				return nil, perr
			}

			c = goredis.NewClient(opt)
		default:
			c = goredis.NewClient(&goredis.Options{Addr: addr})
		}

		if err = c.Ping(context.Background()).Err(); err != nil {
			_ = c.Close()

			return nil, err
		}

		s.owned = append(s.owned, c)

		return c, nil
	}

	if s.primary == nil {
		if cfg.url == "" && cfg.addr == "" {
			return nil, errors.New("redis: no client: use WithClient, WithURL, WithAddr, " +
				EnvURL + " or " + EnvAddr)
		}

		c, err := open(cfg.url, cfg.addr)
		if err != nil {
			return nil, fmt.Errorf("redis: primary: %w", err)
		}

		s.primary = c
	}

	if s.replica == nil && (cfg.replicaURL != "" || cfg.replicaAddr != "") {
		c, err := open(cfg.replicaURL, cfg.replicaAddr)
		if err != nil {
			_ = s.Close()

			return nil, fmt.Errorf("redis: replica: %w", err)
		}

		s.replica = c
	}

	return s, nil
}

func MustNew(opts ...Option) *Store {
	return rkit.Must(New(opts...))
}

// Close closes clients this store opened. Clients passed with WithClient or
// WithReplicaClient stay open.
func (s *Store) Close() error {
	var err error

	for _, c := range s.owned {
		err = errors.Join(err, c.Close())
	}

	s.owned = nil

	return err
}

func (s *Store) Outbox() store.Outbox { return s }

func (s *Store) Tx(ctx context.Context, fn func(ctx context.Context, tx store.Tx) error) error {
	t := newTx(s, s.primary, false)
	if err := fn(ctx, t); err != nil {
		return err
	}

	return t.commit(ctx)
}

// View runs fn against a read-only transaction. When a replica is configured
// and ctx does not carry store.WithPrimary, the read uses the replica.
func (s *Store) View(ctx context.Context, fn func(ctx context.Context, tx store.Tx) error) error {
	return fn(ctx, newTx(s, s.readClient(ctx), true))
}

func (s *Store) readClient(ctx context.Context) goredis.Cmdable {
	if s.replica == nil || store.PrimaryRead(ctx) {
		return s.primary
	}

	return s.replica
}

func (s *Store) stateKey(kind, id string) string {
	return s.prefix + ":s:" + token(kind) + ":" + token(id)
}

func (s *Store) kindKey(kind string) string {
	return s.prefix + ":k:" + token(kind)
}

func (s *Store) itemKey(id string) string {
	return s.prefix + ":o:" + id
}

func (s *Store) seqKey() string     { return s.prefix + ":seq" }
func (s *Store) pendingKey() string { return s.prefix + ":pending" }

// token makes kind and id safe to embed in one key without colliding.
func token(s string) string {
	return strconv.FormatUint(uint64(len(s)), 16) + ":" + s
}

// load reads one record. ok is false when it is absent.
func (s *Store) load(ctx context.Context, c goredis.Cmdable, kind, id string) (store.Record, bool, error) {
	vals, err := c.HMGet(ctx, s.stateKey(kind, id), "v", "d").Result()
	if err != nil {
		return store.Record{}, false, err
	}

	if vals[0] == nil {
		return store.Record{}, false, nil
	}

	ver, err := strconv.ParseUint(asString(vals[0]), 10, 64)
	if err != nil {
		return store.Record{}, false, fmt.Errorf("redis: version %s/%s: %w", kind, id, err)
	}

	return store.Record{Kind: kind, ID: id, Version: ver, Data: []byte(asString(vals[1]))}, true, nil
}

// OutboxStats reports the unpublished backlog, including leased entries.
func (s *Store) OutboxStats(ctx context.Context) (store.OutboxStats, error) {
	vals, err := goredis.NewScript(statsLua).Run(ctx, s.primary, []string{s.prefix, s.pendingKey()}).Slice()
	if err != nil {
		return store.OutboxStats{}, err
	}

	if len(vals) != 2 {
		return store.OutboxStats{}, fmt.Errorf("redis: outbox stats: %v", vals)
	}

	st := store.OutboxStats{Pending: int(asInt64(vals[0]))}
	if nano := asInt64(vals[1]); nano > 0 {
		st.Oldest = time.Unix(0, nano)
	}

	return st, nil
}

func (s *Store) Claim(ctx context.Context, limit int, leaseFor time.Duration) ([]store.OutboxEntry, error) {
	if limit <= 0 {
		return nil, nil
	}

	now := time.Now().UnixMilli()

	vals, err := goredis.NewScript(claimLua).Run(
		ctx,
		s.primary,
		[]string{s.prefix, s.pendingKey()},
		now,
		limit,
		leaseFor.Milliseconds(),
	).Slice()
	if errors.Is(err, goredis.Nil) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if len(vals)%4 != 0 {
		return nil, fmt.Errorf("redis: claim: %d fields", len(vals))
	}

	out := make([]store.OutboxEntry, 0, len(vals)/4)
	for i := 0; i < len(vals); i += 4 {
		out = append(out, store.OutboxEntry{
			ID:           asString(vals[i]),
			PartitionKey: asString(vals[i+1]),
			Data:         []byte(asString(vals[i+2])),
			CreatedAt:    time.Unix(0, asInt64(vals[i+3])),
		})
	}

	return out, nil
}

func (s *Store) Ack(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	return goredis.NewScript(ackLua).Run(ctx, s.primary, []string{s.prefix, s.pendingKey()}, args...).Err()
}

func asString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case nil:
		return ""
	default:
		return fmt.Sprint(x)
	}
}

func asInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)

		return n
	default:
		return 0
	}
}
