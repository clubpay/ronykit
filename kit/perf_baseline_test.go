package kit

import (
	"context"
	"sync"
	"testing"
	"time"
)

type benchConn struct {
	kv map[string]string
}

func newBenchConn() *benchConn {
	return &benchConn{kv: map[string]string{}}
}

func (b *benchConn) ConnID() uint64                            { return 1 }
func (b *benchConn) ClientIP() string                          { return "127.0.0.1" }
func (b *benchConn) WriteEnvelope(*Envelope) error             { return nil }
func (b *benchConn) Stream() bool                              { return false }
func (b *benchConn) Walk(fn func(key string, val string) bool) {}
func (b *benchConn) Get(key string) string                     { return b.kv[key] }
func (b *benchConn) Set(key string, val string)                { b.kv[key] = val }

type benchCluster struct{}

func (benchCluster) Start(context.Context) error       { return nil }
func (benchCluster) Shutdown(context.Context) error    { return nil }
func (benchCluster) Subscribe(string, ClusterDelegate) {}
func (benchCluster) Publish(string, []byte) error      { return nil }
func (benchCluster) Subscribers() ([]string, error)    { return []string{"node-A", "node-B"}, nil }

func BenchmarkContextOutSendRaw(b *testing.B) {
	ls := &localStore{kv: map[string]any{}}
	ctx := newContext(ls)
	conn := newBenchConn()
	ctx.conn = conn
	ctx.in = newEnvelope(ctx, conn, false)
	payload := RawMessage("hello-world")

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		ctx.Out().SetMsg(payload).Send()
	}
}

func BenchmarkEnvelopeCarrierFillWithContext(b *testing.B) {
	ls := &localStore{kv: map[string]any{}}
	ctx := newContext(ls)
	conn := newBenchConn()
	conn.Set("client", "A")
	ctx.conn = conn
	ctx.in = newEnvelope(ctx, conn, false)
	ctx.in.SetID("env").SetHdr("x", "y").SetMsg(RawMessage("payload"))
	ctx.setServiceName("svc").setContractID("cid").setRoute("route")
	ctx.sb = &southBridge{}

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = newEnvelopeCarrier(incomingCarrier, "sid", "origin", "target").FillWithContext(ctx)
	}
}

func BenchmarkForwarderHandlerRemoteTimeout(b *testing.B) {
	ls := &localStore{kv: map[string]any{}}
	sb := &southBridge{
		ctxPool:      ctxPool{ls: ls},
		id:           "node-A",
		wg:           &sync.WaitGroup{},
		eh:           func(*Context, error) {},
		cb:           benchCluster{},
		inProgress:   map[string]*clusterConn{},
		msgFactories: map[string]MessageFactoryFunc{},
	}
	h := sb.genForwarderHandler(func(*LimitedContext) (string, error) {
		return "node-B", nil
	})

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		conn := newBenchConn()
		ctx := newContext(ls)
		ctx.conn = conn
		ctx.in = newEnvelope(ctx, conn, false)
		ctx.in.SetMsg(RawMessage(`{"msg":"x"}`))
		ctx.setServiceName("svc").setContractID("cid").setRoute("RPC.Ping")
		ctx.sb = sb
		ctx.rxt = time.Nanosecond

		h(ctx)
	}
}
