package rony

import (
	"errors"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
)

type rawIn struct {
	Payload string `json:"payload"`
}

func TestJoinRESTPath(t *testing.T) {
	tests := []struct {
		base string
		path string
		want string
	}{
		{base: "", path: "/clusters", want: "/clusters"},
		{base: "/v1", path: "/clusters", want: "/v1/clusters"},
		{base: "/v1", path: "clusters", want: "/v1/clusters"},
		{base: "/v1/", path: "clusters", want: "/v1/clusters"},
		{base: "/v1", path: "/clusters/{id}", want: "/v1/clusters/{id}"},
	}

	for _, tc := range tests {
		if got := joinRESTPath(tc.base, tc.path); got != tc.want {
			t.Fatalf("joinRESTPath(%q, %q) = %q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
}

func TestWithBasePathPrefixesUnaryRoutes(t *testing.T) {
	srv := NewServer()

	handler := func(_ *UnaryCtx[EMPTY, NOP], _ goodIn) (*goodOut, error) {
		return &goodOut{OK: true}, nil
	}

	Setup[EMPTY, NOP](
		srv,
		"svc",
		EmptyState(),
		SetupOptionGroup[EMPTY, NOP](
			WithBasePath[EMPTY, NOP]("/v1"),
			WithUnary[EMPTY, NOP, goodIn, goodOut](
				handler,
				POST("clusters", UnaryName("CreateCluster")),
			),
		),
		WithUnary[EMPTY, NOP, goodIn, goodOut](
			handler,
			GET("/health", UnaryName("Health")),
		),
	)

	svc := srv.cfg.services["svc"]
	if svc == nil || len(svc.Contracts) != 2 {
		t.Fatalf("unexpected contracts: %#v", svc)
	}

	var (
		createPath string
		healthPath string
	)

	for _, contract := range svc.Contracts {
		if len(contract.RouteSelectors) != 1 {
			t.Fatalf("unexpected route selectors: %#v", contract.RouteSelectors)
		}

		rest, ok := contract.RouteSelectors[0].Selector.(interface {
			GetPath() string
		})
		if !ok {
			t.Fatal("expected REST route selector")
		}

		switch contract.RouteSelectors[0].Name {
		case "CreateCluster":
			createPath = rest.GetPath()
		case "Health":
			healthPath = rest.GetPath()
		default:
			t.Fatalf("unexpected route name: %s", contract.RouteSelectors[0].Name)
		}
	}

	if createPath != "/v1/clusters" {
		t.Fatalf("unexpected create path: %s", createPath)
	}
	if healthPath != "/health" {
		t.Fatalf("unexpected health path: %s", healthPath)
	}
}

func TestUnaryOptionHelpers(t *testing.T) {
	cfg := unaryConfig{}

	ALL("/all")(&cfg)
	POST("/post")(&cfg)
	PUT("/put")(&cfg)
	DELETE("/delete")(&cfg)
	PATCH("/patch")(&cfg)
	HEAD("/head")(&cfg)
	OPTIONS("/options")(&cfg)
	REST("GET", "/get",
		UnaryName("Custom"),
		UnaryDecoder(nil),
		UnaryDeprecated(true),
	)(&cfg)

	UnaryInputMeta(desc.WithField("f", desc.FieldMeta{Optional: true}))(&cfg)
	UnaryOutputMeta(desc.WithField("g", desc.FieldMeta{Deprecated: true}))(&cfg)
	UnaryHeader(RequiredHeader("x"), OptionalHeader("y"))(&cfg)
	UnaryMiddleware(func(*kit.Context) {})(&cfg)
	UnaryMiddlewareFn(func() StatelessMiddleware { return func(*kit.Context) {} })(&cfg)

	if len(cfg.Selectors) == 0 {
		t.Fatal("expected selectors to be set")
	}
	if len(cfg.Middlewares) == 0 {
		t.Fatal("expected middlewares to be set")
	}
	if len(cfg.Headers) != 2 {
		t.Fatalf("unexpected headers count: %d", len(cfg.Headers))
	}
	if cfg.InputMetaOptions == nil || cfg.OutputMetaOptions == nil {
		t.Fatal("expected meta options to be set")
	}
}

func TestSetupRawUnary(t *testing.T) {
	srv := NewServer()

	handler := func(_ *UnaryCtx[EMPTY, NOP], in kit.RawMessage) (kit.RawMessage, error) {
		return kit.RawMessage("ok"), nil
	}

	Setup[EMPTY, NOP](
		srv,
		"svc",
		EmptyState(),
		WithRawUnary[EMPTY, NOP, kit.RawMessage](
			handler,
			GET("/raw"),
		),
	)

	svc := srv.cfg.services["svc"]
	if svc == nil || len(svc.Contracts) != 1 {
		t.Fatalf("unexpected contracts: %#v", svc)
	}

	contract := svc.Contracts[0]
	err := kit.NewTestContext().
		Input(kit.RawMessage("in"), kit.EnvelopeHdr{}).
		SetHandler(contract.Handlers...).
		Expect(func(e *kit.Envelope) error {
			if string(e.GetMsg().(kit.RawMessage)) != "ok" { //nolint:forcetypeassert
				return errors.New("unexpected message")
			}

			return nil
		}).
		RunREST()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetupRawUnaryErrorPath(t *testing.T) {
	srv := NewServer()

	handler := func(_ *UnaryCtx[EMPTY, NOP], _ rawIn) (kit.RawMessage, error) {
		return nil, errors.New("boom")
	}

	Setup[EMPTY, NOP](
		srv,
		"svc",
		EmptyState(),
		WithRawUnary[EMPTY, NOP, rawIn](
			handler,
			GET("/raw"),
		),
	)

	svc := srv.cfg.services["svc"]
	contract := svc.Contracts[0]
	err := kit.NewTestContext().
		Input(&rawIn{Payload: "x"}, kit.EnvelopeHdr{}).
		SetHandler(contract.Handlers...).
		Receiver(func(out ...*kit.Envelope) error {
			if len(out) == 0 {
				return errors.New("expected error response")
			}

			return nil
		}).
		RunREST()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
