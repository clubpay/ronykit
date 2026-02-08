package kit

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

type outMessage struct {
	OK bool `json:"ok"`
}

type inMessage struct {
	Name string `json:"name"`
}

func TestEdgeOptions(t *testing.T) {
	cfg := &edgeConfig{}
	logger := NOPLogger{}
	trace := &testTracer{}
	del := &testConnDelegate{}

	WithLogger(logger)(cfg)
	WithPrefork()(cfg)
	WithGateway(&testGateway{})(cfg)
	WithCluster(&testCluster{})(cfg)
	WithService(testService{name: "svc"})(cfg)
	WithServiceBuilder(testServiceBuilder{svc: testService{name: "svc2"}})(cfg)
	WithShutdownTimeout(5 * time.Second)(cfg)
	WithErrorHandler(func(_ *Context, _ error) {})(cfg)
	WithGlobalHandlers(func(_ *Context) {})(cfg)
	WithTrace(trace)(cfg)
	WithConnDelegate(del)(cfg)
	ReusePort(true)(cfg)

	if cfg.logger != logger {
		t.Fatal("expected logger to be set")
	}
	if !cfg.prefork {
		t.Fatal("expected prefork to be true")
	}
	if len(cfg.gateways) != 1 {
		t.Fatalf("unexpected gateways count: %d", len(cfg.gateways))
	}
	if cfg.cluster == nil {
		t.Fatal("expected cluster to be set")
	}
	if len(cfg.services) != 2 {
		t.Fatalf("unexpected services count: %d", len(cfg.services))
	}
	if cfg.shutdownTimeout != 5*time.Second {
		t.Fatalf("unexpected shutdown timeout: %v", cfg.shutdownTimeout)
	}
	if cfg.errHandler == nil {
		t.Fatal("expected error handler to be set")
	}
	if len(cfg.globalHandlers) != 1 {
		t.Fatalf("unexpected global handlers count: %d", len(cfg.globalHandlers))
	}
	if cfg.tracer != trace {
		t.Fatal("expected tracer to be set")
	}
	if cfg.connDelegate != del {
		t.Fatal("expected conn delegate to be set")
	}
	if !cfg.reusePort {
		t.Fatal("expected reusePort to be true")
	}
}

func TestWrapContractOrdering(t *testing.T) {
	h1 := func(*Context) {}
	h2 := func(*Context) {}
	m1 := func(*Envelope) {}
	m2 := func(*Envelope) {}
	m3 := func(*Envelope) {}

	base := &testContract{
		handlers:  []HandlerFunc{h1},
		modifiers: []ModifierFunc{m1},
	}

	wrapped := WrapContract(base, ContractWrapperFunc(func(c Contract) Contract {
		return &contractWrap{
			Contract: c,
			h:        []HandlerFunc{h2},
			preM:     []ModifierFunc{m2},
			postM:    []ModifierFunc{m3},
		}
	}))

	handlers := wrapped.Handlers()
	if len(handlers) != 2 || handlers[0] == nil || handlers[1] == nil {
		t.Fatalf("unexpected handlers: %#v", handlers)
	}
	modifiers := wrapped.Modifiers()
	if len(modifiers) != 3 {
		t.Fatalf("unexpected modifiers: %d", len(modifiers))
	}
}

func TestEdgeServerLifecycle(t *testing.T) {
	gw := &testGateway{}
	cluster := &testCluster{}

	contract := &testContract{
		id:     "c1",
		enc:    JSON,
		input:  &inMessage{},
		output: &outMessage{},
		sel:    testRESTSelector{method: "GET", path: "/v1", encoding: JSON},
		handlers: []HandlerFunc{
			func(ctx *Context) {
				ctx.Out().SetMsg(&outMessage{OK: true}).Send()
			},
		},
	}

	svc := testService{name: "svc", contracts: []Contract{contract}}

	s := NewServer(
		WithGateway(gw),
		WithCluster(cluster),
		WithService(svc),
		WithShutdownTimeout(time.Millisecond*5),
		ReusePort(true),
	)

	s.Start(context.Background())
	if gw.startCalls != 1 {
		t.Fatalf("expected gateway start call, got: %d", gw.startCalls)
	}
	if !gw.lastCfg.ReusePort {
		t.Fatal("expected reuse port to be set")
	}
	if cluster.startCalls != 1 {
		t.Fatalf("expected cluster start call, got: %d", cluster.startCalls)
	}
	if len(gw.regs) != 1 {
		t.Fatalf("unexpected register calls: %d", len(gw.regs))
	}
	if cluster.subscribedID == "" {
		t.Fatal("expected cluster subscription id")
	}

	s.Shutdown(context.Background())
	if gw.shutdownCalls != 1 {
		t.Fatalf("expected gateway shutdown call, got: %d", gw.shutdownCalls)
	}
	if cluster.shutdownCalls != 1 {
		t.Fatalf("expected cluster shutdown call, got: %d", cluster.shutdownCalls)
	}
}

type flushBuffer struct {
	bytes.Buffer
	syncCalls  int
	flushCalls int
}

func (b *flushBuffer) Sync() error {
	b.syncCalls++

	return nil
}

func (b *flushBuffer) Flush() error {
	b.flushCalls++

	return nil
}

type flushOnlyBuffer struct {
	bytes.Buffer
	flushCalls int
}

func (b *flushOnlyBuffer) Flush() error {
	b.flushCalls++

	return nil
}

func TestEdgeServerPrintRoutes(t *testing.T) {
	gw := &testGateway{}

	restContract := &testContract{
		id:     "rest",
		enc:    JSON,
		input:  &inMessage{},
		output: &outMessage{},
		sel:    testRESTSelector{method: "GET", path: "/v1", encoding: JSON},
		handlers: []HandlerFunc{
			func(*Context) {},
		},
	}

	rpcContract := &testContract{
		id:     "rpc",
		enc:    JSON,
		input:  &inMessage{},
		output: &outMessage{},
		sel:    testRPCSelector{predicate: "ping", encoding: JSON},
		handlers: []HandlerFunc{
			func(*Context) {},
		},
	}

	svc := testService{name: "svc", contracts: []Contract{restContract, rpcContract}}

	s := NewServer(WithGateway(gw), WithService(svc))
	buf := &flushBuffer{}
	s.PrintRoutes(buf)
	if buf.syncCalls != 1 {
		t.Fatalf("expected sync call, got: %d", buf.syncCalls)
	}

	bufCompact := &flushBuffer{}
	s.PrintRoutesCompact(bufCompact)
	if bufCompact.syncCalls != 1 {
		t.Fatalf("expected sync call, got: %d", bufCompact.syncCalls)
	}
	if !bytes.Contains(buf.Bytes(), []byte("GET")) {
		t.Fatalf("expected GET route in output, got: %s", buf.String())
	}
	if !bytes.Contains(bufCompact.Bytes(), []byte("ping")) {
		t.Fatalf("expected RPC route in output, got: %s", bufCompact.String())
	}

	flushOnly := &flushOnlyBuffer{}
	s.PrintRoutesCompact(flushOnly)
	if flushOnly.flushCalls != 1 {
		t.Fatalf("expected flush call, got: %d", flushOnly.flushCalls)
	}
}

func TestEdgeServerRegisterServiceDuplicate(t *testing.T) {
	s := NewServer()
	svc := testService{name: "svc", contracts: []Contract{&testContract{id: "svc"}}}
	s.registerService(svc)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate service")
		}
	}()

	s.registerService(svc)
}

func TestLocalStore(t *testing.T) {
	ls := &localStore{kv: map[string]any{}}
	ls.Set("k1", "v1")
	if ls.Get("k1") != "v1" {
		t.Fatalf("unexpected value: %v", ls.Get("k1"))
	}
	if !ls.Exists("k1") {
		t.Fatal("expected key to exist")
	}
	ls.Delete("k1")
	if ls.Exists("k1") {
		t.Fatal("expected key to be deleted")
	}

	ls.Set("pref.a", 1)
	ls.Set("pref.b", 2)
	found := 0
	ls.Scan("pref.", func(key string) bool {
		found++

		return false
	})
	if found != 2 {
		t.Fatalf("unexpected scan count: %d", found)
	}
}

func TestWrapServiceContracts(t *testing.T) {
	base := testService{
		name: "svc",
		contracts: []Contract{
			&testContract{id: "c1"},
		},
	}

	wrapped := WrapServiceContracts(base, ContractWrapperFunc(func(c Contract) Contract {
		return &contractWrap{Contract: c}
	}))
	if wrapped.Name() != "svc" {
		t.Fatalf("unexpected service name: %s", wrapped.Name())
	}
	if len(wrapped.Contracts()) != 1 {
		t.Fatalf("unexpected contracts count: %d", len(wrapped.Contracts()))
	}
}

func TestEdgeServerStartupGatewayError(t *testing.T) {
	gw := &testGateway{startErr: errors.New("boom")}
	s := NewServer(WithGateway(gw))

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on gateway start error")
		}
	}()

	s.startup(context.Background())
}
