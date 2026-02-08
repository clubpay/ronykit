package fasthttp

import (
	"bytes"
	"errors"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/valyala/fasthttp"
)

func TestHTTPConnMethods(t *testing.T) {
	ctx := newRequestCtx(MethodPost, "/path?foo=bar")
	ctx.Request.Header.Set("X-Test", "val")
	ctx.Request.Header.SetHost("example.com")
	ctx.Request.SetBodyRaw([]byte("body"))
	ctx.QueryArgs().Set("q", "1")

	conn := &httpConn{ctx: ctx}

	seenHeader := false
	conn.Walk(func(key, val string) bool {
		if key == "X-Test" && val == "val" {
			seenHeader = true
		}
		return true
	})
	if !seenHeader {
		t.Fatalf("expected to walk headers")
	}

	seenQuery := false
	conn.WalkQueryParams(func(key, val string) bool {
		if key == "q" && val == "1" {
			seenQuery = true
		}
		return true
	})
	if !seenQuery {
		t.Fatalf("expected to walk query params")
	}

	if got := conn.Get("X-Test"); got != "val" {
		t.Fatalf("unexpected header value: %s", got)
	}
	conn.Set("X-Resp", "ok")
	if got := string(ctx.Response.Header.Peek("X-Resp")); got != "ok" {
		t.Fatalf("unexpected response header: %s", got)
	}

	conn.SetStatusCode(fasthttp.StatusCreated)
	if got := ctx.Response.StatusCode(); got != fasthttp.StatusCreated {
		t.Fatalf("unexpected status: %d", got)
	}

	if got := conn.GetHost(); got != "example.com" {
		t.Fatalf("unexpected host: %s", got)
	}
	if got := conn.GetRequestURI(); got != "/path?foo=bar&q=1" {
		t.Fatalf("unexpected request uri: %s", got)
	}
	if got := conn.GetMethod(); got != MethodPost {
		t.Fatalf("unexpected method: %s", got)
	}
	if got := conn.GetPath(); got != "/path" {
		t.Fatalf("unexpected path: %s", got)
	}

	if n, err := conn.Write([]byte("resp")); err != nil || n != 4 {
		t.Fatalf("unexpected write result: %d %v", n, err)
	}
	if !bytes.Contains(ctx.Response.Body(), []byte("resp")) {
		t.Fatalf("unexpected response body: %s", ctx.Response.Body())
	}

	env := newTestEnvelope(newTestContext(conn), conn)
	env.SetID("id").SetHdr("X-Env", "v").SetMsg(kit.RawMessage("env"))
	if err := conn.WriteEnvelope(env); err != nil {
		t.Fatalf("unexpected write envelope error: %v", err)
	}
	if got := string(ctx.Response.Header.Peek("X-Env")); got != "v" {
		t.Fatalf("unexpected envelope header: %s", got)
	}

	if conn.Stream() {
		t.Fatalf("expected non-streaming connection")
	}

	conn.Redirect(fasthttp.StatusFound, "http://example.com/next")
	if got := ctx.Response.StatusCode(); got != fasthttp.StatusFound {
		t.Fatalf("unexpected redirect status: %d", got)
	}
	if loc := string(ctx.Response.Header.Peek(fasthttp.HeaderLocation)); loc == "" {
		t.Fatalf("expected redirect location to be set")
	}
}

func TestHTTPConnGetBodyUncompressed(t *testing.T) {
	body := []byte("hello")

	gzipCtx := newRequestCtx(MethodPost, "/")
	gzipCtx.Request.Header.SetContentEncoding("gzip")
	gzipCtx.Request.SetBodyRaw(fasthttp.AppendGzipBytes(nil, body))
	gzipConn := &httpConn{ctx: gzipCtx}
	got, err := gzipConn.getBodyUncompressed()
	if err != nil || string(got) != "hello" {
		t.Fatalf("unexpected gzip body: %s %v", string(got), err)
	}

	deflateCtx := newRequestCtx(MethodPost, "/")
	deflateCtx.Request.Header.SetContentEncoding("deflate")
	deflateCtx.Request.SetBodyRaw(fasthttp.AppendDeflateBytes(nil, body))
	deflateConn := &httpConn{ctx: deflateCtx}
	got, err = deflateConn.getBodyUncompressed()
	if err != nil || string(got) != "hello" {
		t.Fatalf("unexpected deflate body: %s %v", string(got), err)
	}

	brCtx := newRequestCtx(MethodPost, "/")
	brCtx.Request.Header.SetContentEncoding("br")
	brCtx.Request.SetBodyRaw(fasthttp.AppendBrotliBytes(nil, body))
	brConn := &httpConn{ctx: brCtx}
	got, err = brConn.getBodyUncompressed()
	if err != nil || string(got) != "hello" {
		t.Fatalf("unexpected brotli body: %s %v", string(got), err)
	}

	zstdCtx := newRequestCtx(MethodPost, "/")
	zstdCtx.Request.Header.SetContentEncoding("zstd")
	zstdCtx.Request.SetBodyRaw(fasthttp.AppendZstdBytes(nil, body))
	zstdConn := &httpConn{ctx: zstdCtx}
	got, err = zstdConn.getBodyUncompressed()
	if err != nil || string(got) != "hello" {
		t.Fatalf("unexpected zstd body: %s %v", string(got), err)
	}

	plainCtx := newRequestCtx(MethodPost, "/")
	plainCtx.Request.SetBodyRaw(body)
	plainConn := &httpConn{ctx: plainCtx}
	got, err = plainConn.getBodyUncompressed()
	if err != nil || string(got) != "hello" {
		t.Fatalf("unexpected plain body: %s %v", string(got), err)
	}

	badCtx := newRequestCtx(MethodPost, "/")
	badCtx.Request.Header.SetContentEncoding("snappy")
	badCtx.Request.SetBodyRaw([]byte("bad"))
	badConn := &httpConn{ctx: badCtx}
	if _, err := badConn.getBodyUncompressed(); err == nil {
		t.Fatalf("expected unsupported encoding error")
	}
}

func TestWSConnClosedWrite(t *testing.T) {
	w := &wsConn{kv: map[string]string{}}
	w.Set("k", "v")
	if got := w.Get("k"); got != "v" {
		t.Fatalf("unexpected value: %s", got)
	}
	seen := false
	w.Walk(func(key string, val string) bool {
		if key == "k" && val == "v" {
			seen = true
		}
		return true
	})
	if !seen {
		t.Fatalf("expected Walk to see kv")
	}

	if _, err := w.Write([]byte("x")); !errors.Is(err, kit.ErrWriteToClosedConn) {
		t.Fatalf("expected write to closed conn error, got %v", err)
	}
}
