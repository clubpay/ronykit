package rony

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/clubpay/ronykit/stub/stubgen"
)

func TestServerInitEdgeAndPrintRoutes(t *testing.T) {
	srv := NewServer(
		WithServerName("demo"),
		WithVersion("v1"),
		Listen("127.0.0.1:0"),
		WithAPIDocs("/docs"),
		UseScalarUI(),
	)

	handler := func(_ *UnaryCtx[*goodState, goodAction], _ goodIn) (*goodOut, error) {
		return &goodOut{OK: true}, nil
	}

	Setup[*goodState, goodAction](
		srv,
		"svc",
		func() *goodState { return &goodState{} },
		WithUnary[*goodState, goodAction, goodIn, goodOut](
			handler,
			GET("/v1"),
		),
	)

	if err := srv.initEdge(); err != nil {
		t.Fatalf("initEdge failed: %v", err)
	}

	var buf bytes.Buffer
	srv.PrintRoutes(&buf)
	srv.PrintRoutesCompact(&buf)
	if buf.Len() == 0 {
		t.Fatal("expected routes output")
	}
}

func TestServerStartStopRunAndDocs(t *testing.T) {
	srv := NewServer(
		WithServerName("demo"),
		WithVersion("v1"),
		Listen("127.0.0.1:0"),
	)

	if err := srv.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	srv.Stop(context.Background())

	if err := srv.Run(context.Background()); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	var buf bytes.Buffer
	if err := srv.GenDoc(context.Background(), &buf); err != nil {
		t.Fatalf("GenDoc failed: %v", err)
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "doc.json")
	if err := srv.GenDocFile(context.Background(), path); err != nil {
		t.Fatalf("GenDocFile failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected doc file: %v", err)
	}
}

func TestGenerateStub(t *testing.T) {
	tmpDir := t.TempDir()

	handler := func(_ *UnaryCtx[*goodState, goodAction], _ goodIn) (*goodOut, error) {
		return &goodOut{OK: true}, nil
	}

	err := GenerateStub[*goodState, goodAction](
		"svc",
		"stub",
		tmpDir,
		stubgen.GenFunc(func(in *stubgen.Input) ([]stubgen.GeneratedFile, error) {
			return []stubgen.GeneratedFile{
				{
					Filename: "stub.txt",
					Data:     []byte("ok"),
				},
			}, nil
		}),
		WithUnary[*goodState, goodAction, goodIn, goodOut](
			handler,
			GET("/v1"),
		),
	)
	if err != nil {
		t.Fatalf("GenerateStub failed: %v", err)
	}

	out := filepath.Join(tmpDir, "stub", "stub.txt")
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected stub file: %v", err)
	}
}

func TestServerOptionsAffectConfig(t *testing.T) {
	cfg := defaultServerConfig()
	if len(cfg.gatewayOpts) == 0 {
		t.Fatal("expected default gateway options")
	}
	if cfg.getService("svc").Name != "svc" {
		t.Fatalf("unexpected service name: %s", cfg.getService("svc").Name)
	}
	if len(cfg.allServiceBuilders()) != 1 {
		t.Fatalf("unexpected service builders count: %d", len(cfg.allServiceBuilders()))
	}
	if len(cfg.allServiceDesc()) != 1 {
		t.Fatalf("unexpected service desc count: %d", len(cfg.allServiceDesc()))
	}
	if cfg.Gateway() == nil {
		t.Fatal("expected gateway to be created")
	}
}
