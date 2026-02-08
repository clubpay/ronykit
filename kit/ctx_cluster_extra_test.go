package kit

import (
	"context"
	"testing"
	"time"
)

type testStore struct{}

func (testStore) Set(context.Context, string, string, time.Duration) error { return nil }
func (testStore) SetMulti(context.Context, map[string]string, time.Duration) error {
	return nil
}
func (testStore) Delete(context.Context, string) error                  { return nil }
func (testStore) Get(context.Context, string) (string, error)           { return "", nil }
func (testStore) Scan(context.Context, string, func(string) bool) error { return nil }
func (testStore) ScanWithValue(context.Context, string, func(string, string) bool) error {
	return nil
}

type ctxTestCluster struct {
	subs []string
}

func (ctxTestCluster) Start(context.Context) error       { return nil }
func (ctxTestCluster) Shutdown(context.Context) error    { return nil }
func (ctxTestCluster) Subscribe(string, ClusterDelegate) {}
func (ctxTestCluster) Publish(string, []byte) error      { return nil }
func (t ctxTestCluster) Subscribers() ([]string, error)  { return t.subs, nil }

type ctxTestClusterWithStore struct {
	ctxTestCluster
	store ClusterStore
}

func (t ctxTestClusterWithStore) Store() ClusterStore { return t.store }

func TestContextClusterHelpers(t *testing.T) {
	ctx := NewContext(nil)
	if ctx.HasCluster() {
		t.Fatal("expected no cluster by default")
	}
	if ctx.ClusterID() != "" {
		t.Fatalf("unexpected cluster id: %s", ctx.ClusterID())
	}
	if _, err := ctx.ClusterMembers(); err != ErrClusterNotSet {
		t.Fatalf("expected ErrClusterNotSet, got: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic from ClusterStore when cluster is unset")
		}
	}()
	_ = ctx.ClusterStore()
}

func TestContextClusterStoreAndMembers(t *testing.T) {
	ctx := NewContext(nil)
	cluster := ctxTestCluster{subs: []string{"a", "b"}}
	ctx.sb = &southBridge{cb: cluster, id: "node"}

	if !ctx.HasCluster() {
		t.Fatal("expected cluster to be set")
	}
	if ctx.ClusterID() != "node" {
		t.Fatalf("unexpected cluster id: %s", ctx.ClusterID())
	}
	members, err := ctx.ClusterMembers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("unexpected members count: %d", len(members))
	}
	if ctx.ClusterStore() != nil {
		t.Fatal("expected nil ClusterStore for cluster without store")
	}

	store := testStore{}
	ctx.sb.cb = ctxTestClusterWithStore{ctxTestCluster: cluster, store: store}
	if ctx.ClusterStore() != store {
		t.Fatal("expected cluster store to be returned")
	}
}

func TestContextWithValue(t *testing.T) {
	ctx := NewContext(nil)
	out := ContextWithValue(ctx.Context(), "k", "v")
	if ctx.Get("k") != "v" {
		t.Fatalf("unexpected kit context value: %v", ctx.Get("k"))
	}
	if out.Value("k") != "v" {
		t.Fatalf("unexpected returned context value: %v", out.Value("k"))
	}

	plain := ContextWithValue(context.Background(), "x", "y")
	if plain.Value("x") != "y" {
		t.Fatalf("unexpected plain context value: %v", plain.Value("x"))
	}
}
