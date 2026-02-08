package stub

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/valyala/fasthttp"
)

type echoResponse struct {
	Path   string            `json:"path"`
	Query  string            `json:"query"`
	Method string            `json:"method"`
	Body   string            `json:"body"`
	Header map[string]string `json:"header"`
}

func TestRESTCtxMethodsAndRun(t *testing.T) {
	host, stop := startFastHTTPServer(t, func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/gzip":
			ctx.Response.Header.SetContentEncoding("gzip")
			_, _ = fasthttp.WriteGzip(ctx.Response.BodyWriter(), []byte("hello"))
		case "/deflate":
			ctx.Response.Header.SetContentEncoding("deflate")
			_, _ = fasthttp.WriteDeflate(ctx.Response.BodyWriter(), []byte("world"))
		default:
			resp := echoResponse{
				Path:   string(ctx.Path()),
				Query:  string(ctx.URI().QueryString()),
				Method: string(ctx.Method()),
				Body:   string(ctx.PostBody()),
				Header: map[string]string{"X-Resp": string(ctx.Request.Header.Peek("X-Req"))},
			}
			ctx.Response.Header.Set("X-Resp", "ok")
			ctx.SetStatusCode(http.StatusCreated)
			data, _ := json.Marshal(resp)
			ctx.SetBody(data)
		}
	})
	defer stop()

	s := New(host, Name("stub-test"))
	rest := s.REST(
		WithHeader("X-Req", "yes"),
		WithHeaderMap(map[string]string{"X-Req": "yes"}),
		WithPreflightREST(func(r *fasthttp.Request) {
			r.Header.Set("X-Req", "yes")
		}),
	)

	rest.SetPathF("/items/%d", 10)
	if string(rest.uri.Path()) != "/items/10" {
		t.Fatalf("unexpected path: %s", rest.uri.Path())
	}

	rest.GET("/items/10")
	if string(rest.req.Header.Method()) != http.MethodGet {
		t.Fatalf("unexpected method: %s", rest.req.Header.Method())
	}

	rest.SetQueryMap(map[string]string{"a": "b"})
	rest.SetQuery("b", "c")
	rest.AppendQuery("b", "d")
	rest.SetHeaderMap(map[string]string{"X-Req": "yes"})
	rest.SetContentEncoding("identity")
	rest.SetBody([]byte("payload"))

	rest.SetGZipBody([]byte("gzip"))
	if string(rest.req.Header.ContentEncoding()) != "gzip" {
		t.Fatalf("expected gzip content encoding")
	}

	rest.SetGZipBodyWithLevel([]byte("gzip2"), CompressBestCompression)
	if string(rest.req.Header.ContentEncoding()) != "gzip" {
		t.Fatalf("expected gzip content encoding after level set")
	}

	rest.SetDeflateBodyWithLevel([]byte("deflate"), CompressBestSpeed)
	if string(rest.req.Header.ContentEncoding()) != "deflate" {
		t.Fatalf("expected deflate content encoding")
	}

	frm := &multipart.Form{Value: map[string][]string{"a": {"b"}}}
	rest.SetMultipartForm(frm, "boundary")
	if rest.err != nil {
		t.Fatalf("unexpected multipart error: %v", rest.err)
	}

	rest.SetOKHandler(func(_ context.Context, r RESTResponse) *Error {
		if r.StatusCode() != http.StatusCreated {
			return NewError(http.StatusInternalServerError, "status mismatch")
		}

		return nil
	})

	rest.DefaultResponseHandler(func(_ context.Context, _ RESTResponse) *Error {
		return NewError(http.StatusInternalServerError, "default handler")
	})

	rest.Run(context.Background())
	defer rest.Release()

	if rest.Err() != nil {
		t.Fatalf("unexpected rest error: %v", rest.Err())
	}
	if rest.Error() != nil {
		t.Fatalf("unexpected rest error via Error(): %v", rest.Error())
	}
	if rest.GetHeader("X-Resp") != "ok" {
		t.Fatalf("unexpected response header: %s", rest.GetHeader("X-Resp"))
	}
	if len(rest.DumpRequest()) == 0 || len(rest.DumpResponse()) == 0 {
		t.Fatal("expected dump output")
	}

	var dst bytes.Buffer
	rest.ReadResponseBody(&dst)
	if dst.Len() == 0 {
		t.Fatal("expected response body read")
	}
}

func TestRESTCtxCompressedResponses(t *testing.T) {
	host, stop := startFastHTTPServer(t, func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/gzip":
			ctx.Response.Header.SetContentEncoding("gzip")
			_, _ = fasthttp.WriteGzip(ctx.Response.BodyWriter(), []byte("hello"))
		case "/deflate":
			ctx.Response.Header.SetContentEncoding("deflate")
			_, _ = fasthttp.WriteDeflate(ctx.Response.BodyWriter(), []byte("world"))
		default:
			ctx.SetBodyString("plain")
		}
	})
	defer stop()

	ctx := New(host).REST().GET("/gzip").Run(context.Background())
	defer ctx.Release()

	body, err := ctx.GetUncompressedBody()
	if err != nil || string(body) != "hello" {
		t.Fatalf("unexpected gzip body: %v %s", err, string(body))
	}

	var dst bytes.Buffer
	ctx.ReadUncompressedResponseBody(&dst)
	if dst.String() != "hello" {
		t.Fatalf("unexpected uncompressed body: %s", dst.String())
	}

	ctx = New(host).REST().GET("/deflate").Run(context.Background())
	defer ctx.Release()

	body, err = ctx.GetUncompressedBody()
	if err != nil || string(body) != "world" {
		t.Fatalf("unexpected deflate body: %v %s", err, string(body))
	}

	copyBody := ctx.CopyBody(nil)
	if len(copyBody) == 0 {
		t.Fatal("expected copied body")
	}
}

func TestRESTCtxAutoRunGETExtra(t *testing.T) {
	host, stop := startFastHTTPServer(t, func(ctx *fasthttp.RequestCtx) {
		resp := echoResponse{
			Path:   string(ctx.Path()),
			Query:  string(ctx.URI().QueryString()),
			Method: string(ctx.Method()),
		}
		data, _ := json.Marshal(resp)
		ctx.SetBody(data)
	})
	defer stop()

	type sampleReq struct {
		ID     int      `json:"id"`
		Name   string   `json:"name"`
		Tags   []string `json:"tags"`
		Values []int64  `json:"values"`
	}

	rest := New(host).REST()
	rest.SetMethod(http.MethodGet)
	rest.AutoRun(context.Background(), "/items/{id}", kit.JSON, &sampleReq{
		ID:     42,
		Name:   "alpha",
		Tags:   []string{"a", "b"},
		Values: []int64{1, 2},
	})
	defer rest.Release()

	var resp echoResponse
	if err := json.Unmarshal(rest.GetBody(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Path != "/items/42" {
		t.Fatalf("unexpected path: %s", resp.Path)
	}
	if !strings.Contains(resp.Query, "name=alpha") {
		t.Fatalf("unexpected query: %s", resp.Query)
	}
	if !strings.Contains(resp.Query, "tags=a") || !strings.Contains(resp.Query, "tags=b") {
		t.Fatalf("unexpected tags query: %s", resp.Query)
	}
	if !strings.Contains(resp.Query, "values=1") || !strings.Contains(resp.Query, "values=2") {
		t.Fatalf("unexpected values query: %s", resp.Query)
	}
}

func TestRESTCtxBodyErrors(t *testing.T) {
	rest := New("localhost:0").REST()
	rest.SetBodyErr(nil, kit.ErrExpectationsDontMatch)
	if rest.Err() == nil {
		t.Fatal("expected error from SetBodyErr")
	}
	if rest.Error() == nil {
		t.Fatal("expected error from Error()")
	}
}

func TestRESTCtxHTTPMethods(t *testing.T) {
	rest := New("localhost:0").REST()
	rest.POST("/post")
	if string(rest.req.Header.Method()) != http.MethodPost {
		t.Fatalf("unexpected method: %s", rest.req.Header.Method())
	}
	rest.PUT("/put")
	if string(rest.req.Header.Method()) != http.MethodPut {
		t.Fatalf("unexpected method: %s", rest.req.Header.Method())
	}
	rest.PATCH("/patch")
	if string(rest.req.Header.Method()) != http.MethodPatch {
		t.Fatalf("unexpected method: %s", rest.req.Header.Method())
	}
	rest.OPTIONS("/opt")
	if string(rest.req.Header.Method()) != http.MethodOptions {
		t.Fatalf("unexpected method: %s", rest.req.Header.Method())
	}
}

func TestRESTTraceCarrier(t *testing.T) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	carrier := restTraceCarrier{r: &req.Header}
	carrier.Set("x-trace", "1")
	if carrier.Get("x-trace") != "1" {
		t.Fatalf("unexpected trace value")
	}
}

func startFastHTTPServer(t *testing.T, handler fasthttp.RequestHandler) (string, func()) {
	t.Helper()

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	server := &fasthttp.Server{
		Handler: handler,
	}

	go func() {
		_ = server.Serve(ln)
	}()

	return ln.Addr().String(), func() {
		_ = server.Shutdown()
		_ = ln.Close()
	}
}
