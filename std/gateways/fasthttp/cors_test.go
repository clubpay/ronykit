package fasthttp

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestCORSHandleOptions(t *testing.T) {
	c := newCORS(CORSConfig{
		AllowedOrigins: []string{"https://example.com"},
		AllowedMethods: []string{fasthttp.MethodGet},
	})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodOptions)
	ctx.Request.Header.Set(fasthttp.HeaderOrigin, "https://example.com")
	ctx.Request.Header.Set(fasthttp.HeaderAccessControlRequestHeaders, "X-Token")

	c.handle(ctx)

	if got := string(ctx.Response.Header.Peek(fasthttp.HeaderAccessControlAllowOrigin)); got != "https://example.com" {
		t.Fatalf("unexpected allow origin: %s", got)
	}
	if got := string(ctx.Response.Header.Peek(fasthttp.HeaderAccessControlAllowHeaders)); got != "X-Token" {
		t.Fatalf("unexpected allow headers: %s", got)
	}
	if got := ctx.Response.StatusCode(); got != fasthttp.StatusNoContent {
		t.Fatalf("unexpected status code: %d", got)
	}
}

func TestCORSHandleWS(t *testing.T) {
	c := newCORS(CORSConfig{
		AllowedOrigins:    []string{"https://example.com"},
		IgnoreEmptyOrigin: true,
	})

	ctx := &fasthttp.RequestCtx{}
	if !c.handleWS(ctx) {
		t.Fatal("expected empty origin to be allowed")
	}

	ctx.Request.Header.Set(fasthttp.HeaderOrigin, "https://other.com")
	if c.handleWS(ctx) {
		t.Fatal("expected origin to be rejected")
	}

	ctx.Request.Header.Set(fasthttp.HeaderOrigin, "https://example.com")
	if !c.handleWS(ctx) {
		t.Fatal("expected origin to be allowed")
	}
}
