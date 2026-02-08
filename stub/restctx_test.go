package stub

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/valyala/fasthttp"
)

type testServer struct {
	addr string
	srv  *fasthttp.Server
	ln   net.Listener
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	s := &testServer{}
	s.srv = &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			switch string(ctx.Path()) {
			case "/created":
				ctx.SetStatusCode(http.StatusCreated)
				ctx.SetBody([]byte("created"))
			case "/teapot":
				ctx.SetStatusCode(http.StatusTeapot)
				ctx.SetBody([]byte("short"))
			case "/gzip":
				var buf bytes.Buffer
				zw := gzip.NewWriter(&buf)
				_, _ = zw.Write([]byte("compressed"))
				_ = zw.Close()
				ctx.Response.Header.Set(fasthttp.HeaderContentEncoding, "gzip")
				ctx.SetStatusCode(http.StatusOK)
				ctx.SetBody(buf.Bytes())
			default:
				ctx.SetStatusCode(http.StatusNotFound)
			}
		},
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	s.addr = ln.Addr().String()
	s.ln = ln

	go func() {
		_ = s.srv.Serve(ln)
	}()

	t.Cleanup(func() {
		_ = s.ln.Close()
		_ = s.srv.Shutdown()
	})

	return s
}

func TestHTTPParsing(t *testing.T) {
	ctx, err := HTTP("https://example.com/some")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(ctx.uri.Scheme()); got != "https" {
		t.Fatalf("unexpected scheme: %s", got)
	}
	if got := string(ctx.uri.Path()); got != "/some" {
		t.Fatalf("unexpected path: %s", got)
	}

	if _, err := HTTP("ftp://example.com"); !errors.Is(err, errUnsupportedScheme) {
		t.Fatalf("expected unsupported scheme error, got: %v", err)
	}
	if _, err := HTTP("\t::bad"); err == nil {
		t.Fatal("expected invalid URL error")
	}
}

func TestRESTCtxBodyHelpers(t *testing.T) {
	s := New("example.com")
	rest := s.REST()
	defer rest.Release()

	rest.SetGZipBody([]byte("payload"))
	if got := string(rest.req.Header.Peek(fasthttp.HeaderContentEncoding)); got != "gzip" {
		t.Fatalf("unexpected gzip header: %s", got)
	}

	rest.SetDeflateBody([]byte("payload"))
	if got := string(rest.req.Header.Peek(fasthttp.HeaderContentEncoding)); got != "deflate" {
		t.Fatalf("unexpected deflate header: %s", got)
	}

	rest.SetBodyErr(nil, errors.New("boom"))
	if rest.Err() == nil {
		t.Fatal("expected error after SetBodyErr")
	}
}

func TestRESTCtxHandlersAndGzipResponse(t *testing.T) {
	srv := newTestServer(t)

	s := New(srv.addr)

	var okHandled bool
	restOK := s.REST().
		SetMethod(http.MethodGet).
		SetPath("/created").
		SetOKHandler(func(_ context.Context, r RESTResponse) *Error {
			okHandled = true
			if got := string(r.GetBody()); got != "created" {
				t.Errorf("unexpected body: %s", got)
			}

			return nil
		})
	restOK.Run(context.Background())
	if restOK.Err() != nil {
		t.Fatalf("unexpected error: %v", restOK.Err())
	}
	restOK.Release()
	if !okHandled {
		t.Fatal("expected OK handler to run")
	}

	var defaultHandled bool
	restDefault := s.REST().
		SetMethod(http.MethodGet).
		SetPath("/teapot").
		DefaultResponseHandler(func(_ context.Context, r RESTResponse) *Error {
			defaultHandled = true
			if r.StatusCode() != http.StatusTeapot {
				t.Fatalf("unexpected status: %d", r.StatusCode())
			}

			return nil
		})
	restDefault.Run(context.Background())
	if restDefault.Err() != nil {
		t.Fatalf("unexpected error: %v", restDefault.Err())
	}
	restDefault.Release()
	if !defaultHandled {
		t.Fatal("expected default handler to run")
	}

	restGzip := s.REST().
		SetMethod(http.MethodGet).
		SetPath("/gzip")
	restGzip.Run(context.Background())
	defer restGzip.Release()
	if restGzip.Err() != nil {
		t.Fatalf("unexpected error: %v", restGzip.Err())
	}

	body, err := restGzip.GetUncompressedBody()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != "compressed" {
		t.Fatalf("unexpected uncompressed body: %s", string(body))
	}
}

func TestRESTCtxAutoRunGET(t *testing.T) {
	srv := &fasthttp.Server{}
	srv.Handler = func(ctx *fasthttp.RequestCtx) {
		if string(ctx.Path()) != "/items/42" {
			ctx.SetStatusCode(http.StatusBadRequest)

			return
		}

		args := map[string][]string{}
		ctx.QueryArgs().VisitAll(func(k, v []byte) {
			key := string(k)
			args[key] = append(args[key], string(v))
		})

		if got := args["name"]; len(got) != 1 || got[0] != "widget" {
			ctx.SetStatusCode(http.StatusBadRequest)

			return
		}
		if got := args["tags"]; len(got) != 2 || !(got[0] == "a" && got[1] == "b" || got[0] == "b" && got[1] == "a") {
			ctx.SetStatusCode(http.StatusBadRequest)

			return
		}

		ctx.SetStatusCode(http.StatusOK)
		ctx.SetBody([]byte("ok"))
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	addr := ln.Addr().String()

	go func() {
		_ = srv.Serve(ln)
	}()

	defer func() {
		_ = ln.Close()
		_ = srv.Shutdown()
	}()

	s := New(addr)
	handled := false
	rest := s.REST().SetMethod(http.MethodGet)
	rest.SetResponseHandler(http.StatusOK, func(_ context.Context, r RESTResponse) *Error {
		handled = true

		return nil
	}).DefaultResponseHandler(func(_ context.Context, r RESTResponse) *Error {
		return NewError(r.StatusCode(), "unexpected status")
	})
	defer rest.Release()

	rest.AutoRun(context.Background(), "/items/{id}", kit.JSON, autoReq{
		ID:   42,
		Name: "widget",
		Tags: []string{"a", "b"},
	})

	if rest.Err() != nil {
		t.Fatalf("unexpected error: %v", rest.Err())
	}
	if !handled {
		t.Fatal("expected response handler to run")
	}
}

type autoReq struct {
	ID   int      `json:"id"`
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func TestRESTCtxReadResponseBody(t *testing.T) {
	srv := newTestServer(t)
	s := New(srv.addr)
	rest := s.REST().SetMethod(http.MethodGet).SetPath("/created")
	defer rest.Release()

	rest.Run(context.Background())
	if rest.Err() != nil {
		t.Fatalf("unexpected error: %v", rest.Err())
	}

	var buf bytes.Buffer
	rest.ReadResponseBody(&buf)
	if rest.Err() != nil {
		t.Fatalf("unexpected error: %v", rest.Err())
	}
	if got := buf.String(); got != "created" {
		t.Fatalf("unexpected response body: %s", got)
	}
}

func TestRESTCtxReadUncompressedResponseBody(t *testing.T) {
	srv := newTestServer(t)
	s := New(srv.addr)
	rest := s.REST().SetMethod(http.MethodGet).SetPath("/gzip")
	defer rest.Release()

	rest.Run(context.Background())
	if rest.Err() != nil {
		t.Fatalf("unexpected error: %v", rest.Err())
	}

	var buf bytes.Buffer
	rest.ReadUncompressedResponseBody(&buf)
	if rest.Err() != nil {
		t.Fatalf("unexpected error: %v", rest.Err())
	}
	if got := buf.String(); got != "compressed" {
		t.Fatalf("unexpected response body: %s", got)
	}
}

func TestRESTCtxCopyBody(t *testing.T) {
	srv := newTestServer(t)
	s := New(srv.addr)
	rest := s.REST().SetMethod(http.MethodGet).SetPath("/created")
	defer rest.Release()

	rest.Run(context.Background())
	if rest.Err() != nil {
		t.Fatalf("unexpected error: %v", rest.Err())
	}

	copied := rest.CopyBody(nil)
	if string(copied) != "created" {
		t.Fatalf("unexpected copied body: %s", string(copied))
	}

	rest.res.SetBody([]byte("changed"))
	if string(copied) != "created" {
		t.Fatalf("copied body mutated: %s", string(copied))
	}
}

func TestRESTCtxDumpRequestAndResponse(t *testing.T) {
	srv := newTestServer(t)
	s := New(srv.addr)
	rest := s.REST().SetMethod(http.MethodGet).SetPath("/created")
	defer rest.Release()

	var reqBuf, resBuf bytes.Buffer
	rest.DumpRequestTo(&reqBuf)
	rest.DumpResponseTo(&resBuf)
	rest.Run(context.Background())

	if rest.Err() != nil {
		t.Fatalf("unexpected error: %v", rest.Err())
	}
	if reqBuf.Len() == 0 {
		t.Fatal("expected request dump output")
	}
	if resBuf.Len() == 0 {
		t.Fatal("expected response dump output")
	}
}

func TestRESTCtxSetMultipartForm(t *testing.T) {
	frm := &multipart.Form{
		Value: map[string][]string{"key": {"value"}},
	}

	rest := New("example.com").REST()
	defer rest.Release()

	rest.SetMultipartForm(frm, "boundary")
	if rest.Err() != nil {
		t.Fatalf("unexpected error: %v", rest.Err())
	}
}

func TestRESTCtxGetBodyWriter(t *testing.T) {
	rest := New("example.com").REST()
	defer rest.Release()

	w := rest.GetBodyWriter()
	_, err := io.WriteString(w, "body")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(rest.req.Body()); got != "body" {
		t.Fatalf("unexpected body: %s", got)
	}
}
