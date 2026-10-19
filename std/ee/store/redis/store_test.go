package redis_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/store"
	"github.com/clubpay/ronykit/ee/store/storetest"
	"github.com/clubpay/ronykit/std/ee/store/redis"
	"github.com/clubpay/ronykit/x/rkit"
)

// EE_REDIS_TEST_ADDR enables the tests, e.g. localhost:6379.
// EE_REDIS_TEST_REPLICA_ADDR is a second server used only for the routing test.
const (
	envAddr        = "EE_REDIS_TEST_ADDR"
	envReplicaAddr = "EE_REDIS_TEST_REPLICA_ADDR"
)

func newStore(t *testing.T, opts ...redis.Option) *redis.Store {
	t.Helper()

	addr := os.Getenv(envAddr)
	if addr == "" {
		t.Skip(envAddr + " not set")
	}

	prefix := "{t" + rkit.RandomID(10) + "}"
	s, err := redis.New(append([]redis.Option{
		redis.WithAddr(addr),
		redis.WithPrefix(prefix),
	}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = s.Close() })

	return s
}

func TestConformance(t *testing.T) {
	storetest.Run(t, func(t *testing.T) store.Store { return newStore(t) })
}

func TestNewRequiresClient(t *testing.T) {
	t.Setenv(redis.EnvURL, "")
	t.Setenv(redis.EnvAddr, "")
	t.Setenv(redis.EnvReplicaURL, "")
	t.Setenv(redis.EnvReplicaAddr, "")

	if _, err := redis.New(); err == nil {
		t.Fatal("New without a client must fail")
	}
}

func TestReplicaReads(t *testing.T) {
	replica := os.Getenv(envReplicaAddr)
	if replica == "" {
		t.Skip(envReplicaAddr + " not set")
	}

	s := newStore(t, redis.WithReplicaAddr(replica))
	ctx := t.Context()

	err := s.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
		return tx.Insert(ctx, store.Record{Kind: "order", ID: "o1", Data: []byte(`{"id":"o1"}`)})
	})
	if err != nil {
		t.Fatal(err)
	}

	err = s.View(ctx, func(ctx context.Context, tx store.Tx) error {
		_, err := tx.Get(ctx, "order", "o1")

		return err
	})
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("replica read: %v", err)
	}

	err = s.View(ee.WithPrimary(ctx), func(ctx context.Context, tx store.Tx) error {
		_, err := tx.Get(ctx, "order", "o1")

		return err
	})
	if err != nil {
		t.Fatalf("primary read: %v", err)
	}
}
