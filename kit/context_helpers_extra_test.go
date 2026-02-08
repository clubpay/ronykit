package kit

import (
	"context"
	"testing"
	"time"
)

func TestContextKVHelpers(t *testing.T) {
	ls := &localStore{kv: map[string]any{}}
	ctx := newContext(ls)
	ctx.Set("i64", int64(42))
	ctx.Set("i32", int32(7))
	ctx.Set("u64", uint64(9))
	ctx.Set("u32", uint32(3))
	ctx.Set("str", "v")
	ctx.Set("bytes", []byte("b"))

	if !ctx.Exists("i64") {
		t.Fatal("expected Exists to return true")
	}
	if ctx.GetInt64("i64", 0) != 42 {
		t.Fatalf("unexpected int64 value: %d", ctx.GetInt64("i64", 0))
	}
	if ctx.GetInt32("i32", 0) != 7 {
		t.Fatalf("unexpected int32 value: %d", ctx.GetInt32("i32", 0))
	}
	if ctx.GetUint64("u64", 0) != 9 {
		t.Fatalf("unexpected uint64 value: %d", ctx.GetUint64("u64", 0))
	}
	if ctx.GetUint32("u32", 0) != 3 {
		t.Fatalf("unexpected uint32 value: %d", ctx.GetUint32("u32", 0))
	}
	if ctx.GetString("str", "") != "v" {
		t.Fatalf("unexpected string value: %s", ctx.GetString("str", ""))
	}
	if string(ctx.GetBytes("bytes", nil)) != "b" {
		t.Fatalf("unexpected bytes value: %s", string(ctx.GetBytes("bytes", nil)))
	}
	if ctx.GetString("missing", "d") != "d" {
		t.Fatalf("unexpected default value: %s", ctx.GetString("missing", "d"))
	}
	if ctx.LocalStore() != ls {
		t.Fatal("expected local store to be returned")
	}
}

func TestLimitedContext(t *testing.T) {
	ls := &localStore{kv: map[string]any{}}
	ctx := newContext(ls)
	conn := newTestConn()
	ctx.conn = conn
	ctx.in = newEnvelope(ctx, conn, false)
	ctx.in.SetMsg(&inMessage{Name: "x"})
	ctx.setRoute("route").setServiceName("svc")
	ctx.sb = &southBridge{cb: &testCluster{}, id: "node"}

	limited := ctx.Limited()
	if limited.Context() == nil {
		t.Fatal("expected context to be set")
	}
	if limited.In() == nil {
		t.Fatal("expected envelope to be set")
	}
	if limited.Conn() == nil {
		t.Fatal("expected conn to be set")
	}
	if limited.IsREST() {
		t.Fatal("expected non-REST context")
	}

	limited.SetHdr("k", "v")
	limited.SetHdrMap(map[string]string{"x": "y"})
	if ctx.hdr["k"] != "v" || ctx.hdr["x"] != "y" {
		t.Fatalf("unexpected headers: %#v", ctx.hdr)
	}
	if limited.Route() != "route" || limited.ServiceName() != "svc" {
		t.Fatalf("unexpected route/service: %s/%s", limited.Route(), limited.ServiceName())
	}
	if limited.ClusterID() != "node" {
		t.Fatalf("unexpected cluster id: %s", limited.ClusterID())
	}
	members, err := limited.ClusterMembers()
	if err != nil || len(members) != 1 {
		t.Fatalf("unexpected cluster members: %v %#v", err, members)
	}
	if limited.ClusterStore() != nil {
		t.Fatal("expected nil cluster store for cluster without store")
	}

	limited.SetUserContext(context.WithValue(context.Background(), "k", "v"))
	if ctx.Context().Value("k") != "v" {
		t.Fatalf("unexpected user context value: %v", ctx.Context().Value("k"))
	}
	limited.SetRemoteExecutionTimeout(10 * time.Millisecond)
	if ctx.rxt != 10*time.Millisecond {
		t.Fatalf("unexpected remote execution timeout: %v", ctx.rxt)
	}
}
