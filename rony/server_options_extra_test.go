package rony

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"github.com/clubpay/ronykit/kit"
)

type dummyTracer struct{}

func (dummyTracer) Inject(_ context.Context, _ kit.TraceCarrier)                    {}
func (dummyTracer) Extract(ctx context.Context, _ kit.TraceCarrier) context.Context { return ctx }
func (dummyTracer) Handler() kit.HandlerFunc {
	return func(ctx *kit.Context) {
		ctx.Next()
	}
}

func TestServerOptionsApply(t *testing.T) {
	cfg := defaultServerConfig()

	WithServerName("srv")(&cfg)
	WithVersion("1")(&cfg)
	WithCORS(CORSConfig{})(&cfg)
	Listen("127.0.0.1:0")(&cfg)
	WithCompression(CompressionLevelBestSpeed)(&cfg)
	WithPredicateKey("cmd")(&cfg)
	WithWebsocketEndpoint("/ws")(&cfg)
	WithCustomRPC(nil, nil)(&cfg)
	WithTracer(dummyTracer{})(&cfg)
	WithLogger(kit.NOPLogger{})(&cfg)
	WithPrefork()(&cfg)
	WithShutdownTimeout(time.Second)(&cfg)
	WithGlobalHandlers(func(*kit.Context) {})(&cfg)
	WithDisableHeaderNamesNormalizing()(&cfg)
	WithAPIDocs("/docs")(&cfg)
	WithServerFS("/static", ".", fstest.MapFS{"index.html": {Data: []byte("ok")}})(&cfg)
	WithErrorHandler(func(*kit.Context, error) {})(&cfg)
	UseSwaggerUI()(&cfg)
	UseRedocUI()(&cfg)
	UseScalarUI()(&cfg)

	if cfg.serverName != "srv" {
		t.Fatalf("unexpected server name: %s", cfg.serverName)
	}
	if cfg.version != "1" {
		t.Fatalf("unexpected version: %s", cfg.version)
	}
	if cfg.serveDocsPath != "/docs" {
		t.Fatalf("unexpected docs path: %s", cfg.serveDocsPath)
	}
	if cfg.serveDocsUI != scalarUI {
		t.Fatalf("unexpected docs ui: %s", cfg.serveDocsUI)
	}
	if len(cfg.gatewayOpts) == 0 {
		t.Fatal("expected gateway options to be set")
	}
	if len(cfg.edgeOpts) == 0 {
		t.Fatal("expected edge options to be set")
	}
}
