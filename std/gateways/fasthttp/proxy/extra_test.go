package proxy

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

type testLogger struct {
	calls int
	last  string
}

func (l *testLogger) Printf(format string, args ...any) {
	l.calls++
	l.last = fmt.Sprintf(format, args...)
}

func newFasthttpCtx() *fasthttp.RequestCtx {
	var ctx fasthttp.RequestCtx
	ctx.Init(&ctx.Request, &net.TCPAddr{IP: net.ParseIP("10.0.0.1"), Port: 4321}, nil)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("/")
	return &ctx
}

func TestDefaultBuildOption(t *testing.T) {
	opt := defaultBuildOption()
	if opt == nil || opt.logger == nil {
		t.Fatalf("expected default option to be initialized")
	}
	if opt.openBalance || opt.timeout != 0 {
		t.Fatalf("unexpected default option values")
	}
}

func TestBuildOptions(t *testing.T) {
	opt := defaultBuildOption()
	WithAddress("a").apply(opt)
	if len(opt.addresses) != 1 || opt.addresses[0] != "a" {
		t.Fatalf("unexpected address: %v", opt.addresses)
	}

	WithDebug().apply(opt)
	WithTimeout(time.Second).apply(opt)
	WithDisablePathNormalizing(true).apply(opt)
	WithMaxConnDuration(time.Minute).apply(opt)
	if !opt.debug || opt.timeout != time.Second || !opt.disablePathNormalizing || opt.maxConnDuration != time.Minute {
		t.Fatalf("unexpected option values")
	}
	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	WithTLSConfig(tlsCfg).apply(opt)
	if opt.tlsConfig != tlsCfg {
		t.Fatalf("unexpected tls config")
	}

	weights := map[string]Weight{"a": 2, "b": 1}
	WithBalancer(weights).apply(opt)
	if !opt.openBalance || len(opt.addresses) != 2 || len(opt.weights) != 2 {
		t.Fatalf("unexpected balancer setup")
	}
}

func TestWSOptionsAndValidation(t *testing.T) {
	if err := defaultBuildOptionWS().validate(); err == nil {
		t.Fatalf("expected validation error for empty target")
	}

	opt := defaultBuildOptionWS()
	WithURL_OptionWS("ws://example.com").apply(opt)
	WithDebug_OptionWS().apply(opt)
	upgrader := &websocket.FastHTTPUpgrader{}
	WithUpgrader_OptionWS(upgrader).apply(opt)
	dialer := &websocket.Dialer{}
	WithDialer_OptionWS(dialer).apply(opt)
	if err := opt.validate(); err != nil || !opt.debug {
		t.Fatalf("unexpected ws option validation")
	}
	if opt.upgrader != upgrader || opt.dialer != dialer {
		t.Fatalf("unexpected ws option values")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for invalid url")
		}
	}()
	WithURL_OptionWS("://bad").apply(defaultBuildOptionWS())
}

func TestLoggerHelpers(t *testing.T) {
	l := &testLogger{}
	debugf(false, l, "skip")
	if l.calls != 0 {
		t.Fatalf("unexpected debug call")
	}
	debugf(true, l, "hit %d", 1)
	if l.calls != 1 {
		t.Fatalf("expected debug call")
	}
	errorf(l, "err")
	if l.calls != 2 {
		t.Fatalf("expected error call")
	}
	debugf(true, nil, "noop")
	errorf(nil, "noop")
}

func TestChanPoolErrors(t *testing.T) {
	if _, err := NewChanPool(2, 1, func(string) (*ReverseProxy, error) { return nil, nil }); err == nil {
		t.Fatalf("expected invalid capacity error")
	}

	factoryErr := errors.New("factory")
	if _, err := NewChanPool(1, 1, func(string) (*ReverseProxy, error) { return &ReverseProxy{}, factoryErr }); err == nil {
		t.Fatalf("expected factory error")
	}

	factory := func(addr string) (*ReverseProxy, error) { return NewReverseProxyWith(WithAddress(addr)) }
	pool, err := NewChanPool(0, 1, factory)
	if err != nil {
		t.Fatalf("unexpected pool error: %v", err)
	}
	pool.Close()
	if _, err := pool.Get("x"); !errors.Is(err, errClosed) {
		t.Fatalf("expected closed pool error, got %v", err)
	}
	if err := pool.Put(nil); err == nil {
		t.Fatalf("expected nil proxy error")
	}
}

func TestReverseProxyGetClientAndClose(t *testing.T) {
	p, err := NewReverseProxyWith(WithAddress("127.0.0.1:1"))
	if err != nil {
		t.Fatalf("unexpected proxy error: %v", err)
	}
	p.SetClient("127.0.0.1:2")
	if p.getClient().Addr != "127.0.0.1:2" {
		t.Fatalf("unexpected client addr: %s", p.getClient().Addr)
	}
	p.Reset()
	if p.getClient().Addr != "" {
		t.Fatalf("expected reset addr")
	}
	p.Close()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic after close")
		}
	}()
	_ = p.getClient()
}

func TestReverseProxyDoWithTimeout(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer ln.Close()

	srv := fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) { ctx.SetBodyString("ok") },
	}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Shutdown()

	client := &fasthttp.HostClient{Addr: ln.Addr().String()}
	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("http://" + client.Addr + "/")
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	p := &ReverseProxy{opt: &buildOption{timeout: 0}}
	if err := p.doWithTimeout(client, req, res); err != nil {
		t.Fatalf("unexpected do error: %v", err)
	}

	p.opt.timeout = time.Second
	if err := p.doWithTimeout(client, req, res); err != nil {
		t.Fatalf("unexpected do timeout error: %v", err)
	}
}

func TestReverseProxyServeHTTPErrors(t *testing.T) {
	ctx := newFasthttpCtx()
	p := &ReverseProxy{
		opt:     &buildOption{timeout: 0, logger: &nopLogger{}},
		clients: []*fasthttp.HostClient{{Addr: "127.0.0.1:1"}},
	}
	p.ServeHTTP(ctx)
	if ctx.Response.StatusCode() != http.StatusInternalServerError {
		t.Fatalf("expected 500 status, got %d", ctx.Response.StatusCode())
	}

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer ln.Close()
	srv := fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) { time.Sleep(50 * time.Millisecond) },
	}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Shutdown()

	ctx2 := newFasthttpCtx()
	p2 := &ReverseProxy{
		opt:     &buildOption{timeout: 5 * time.Millisecond, logger: &nopLogger{}},
		clients: []*fasthttp.HostClient{{Addr: ln.Addr().String()}},
	}
	p2.ServeHTTP(ctx2)
	if ctx2.Response.StatusCode() != http.StatusRequestTimeout {
		t.Fatalf("expected 408 status, got %d", ctx2.Response.StatusCode())
	}
}

func TestBuiltinForwardHeaderHandler(t *testing.T) {
	ctx := newFasthttpCtx()
	ctx.Request.Header.Set("Origin", "o")
	ctx.Request.Header.Set("Sec-WebSocket-Protocol", "p")
	ctx.Request.Header.Set("Cookie", "c")
	ctx.Request.SetHost("example.com")
	ctx.Request.Header.Set("X-Forwarded-For", "1.1.1.1")

	h := builtinForwardHeaderHandler(ctx)
	if h.Get("Origin") != "o" || h.Get("Sec-WebSocket-Protocol") != "p" || h.Get("Cookie") != "c" {
		t.Fatalf("unexpected forwarded headers: %+v", h)
	}
	if h.Get("Host") != "example.com" {
		t.Fatalf("expected host header")
	}
	if h.Get("X-Forwarded-For") != "1.1.1.1, 10.0.0.1" {
		t.Fatalf("unexpected x-forwarded-for: %s", h.Get("X-Forwarded-For"))
	}
	if h.Get("X-Forwarded-Proto") != "http" {
		t.Fatalf("unexpected proto: %s", h.Get("X-Forwarded-Proto"))
	}
}

type tlsConn struct {
	net.Conn
}

func (t *tlsConn) Handshake() error { return nil }

func (t *tlsConn) ConnectionState() tls.ConnectionState { return tls.ConnectionState{} }

func TestBuiltinForwardHeaderHandlerTLS(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	var ctx fasthttp.RequestCtx
	ctx.Init2(&tlsConn{Conn: c1}, nil, true)
	ctx.Request.SetHost("example.com")
	h := builtinForwardHeaderHandler(&ctx)
	if h.Get("X-Forwarded-Proto") != "https" {
		t.Fatalf("expected https proto")
	}
}

func TestWSCopyResponse(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusForbidden,
		Header:     http.Header{"X-Test": []string{"v"}},
		Body:       io.NopCloser(bytes.NewBufferString("denied")),
	}
	var dst fasthttp.Response
	if err := wsCopyResponse(&dst, resp); err != nil {
		t.Fatalf("unexpected copy error: %v", err)
	}
	if dst.StatusCode() != http.StatusForbidden {
		t.Fatalf("unexpected status: %d", dst.StatusCode())
	}
	if string(dst.Body()) != "denied" {
		t.Fatalf("unexpected body: %s", dst.Body())
	}
	if string(dst.Header.Peek("X-Test")) != "v" {
		t.Fatalf("unexpected header: %s", dst.Header.Peek("X-Test"))
	}
}

func TestWSReverseProxyServeHTTPBadHandshake(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Backend", "ok")
		_, _ = w.Write([]byte("ok"))
	}))
	defer backend.Close()

	wsURL := "ws" + strings.TrimPrefix(backend.URL, "http")
	rp, err := NewWSReverseProxyWith(
		WithURL_OptionWS(wsURL),
		WithForwardHeadersHandlers_OptionWS(func(_ *fasthttp.RequestCtx) http.Header {
			h := make(http.Header)
			h.Set("X-Extra", "1")
			return h
		}),
	)
	if err != nil {
		t.Fatalf("unexpected proxy error: %v", err)
	}

	ctx := newFasthttpCtx()
	rp.ServeHTTP(ctx)
	if ctx.Response.StatusCode() != http.StatusOK {
		t.Fatalf("unexpected status: %d", ctx.Response.StatusCode())
	}
	if string(ctx.Response.Body()) != "ok" {
		t.Fatalf("unexpected body: %s", ctx.Response.Body())
	}
}

func TestWSReverseProxyServeHTTPDialError(t *testing.T) {
	dialer := &websocket.Dialer{
		NetDial: func(string, string) (net.Conn, error) {
			return nil, errors.New("boom")
		},
	}
	rp, err := NewWSReverseProxyWith(
		WithURL_OptionWS("ws://example.com"),
		WithDialer_OptionWS(dialer),
	)
	if err != nil {
		t.Fatalf("unexpected proxy error: %v", err)
	}

	ctx := newFasthttpCtx()
	rp.ServeHTTP(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusServiceUnavailable {
		t.Fatalf("unexpected status: %d", ctx.Response.StatusCode())
	}
}

func TestReplicateWebsocketConnError(t *testing.T) {
	upgrader := websocket.Upgrader{}
	conns := make(chan *websocket.Conn, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conns <- c
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	c1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	c2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	s1 := <-conns
	s2 := <-conns
	defer c1.Close()
	defer c2.Close()
	defer s1.Close()
	defer s2.Close()

	errCh := make(chan error, 1)
	go replicateWebsocketConn(&nopLogger{}, c1, c2, errCh)
	_ = c2.Close()

	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("expected error from replicateWebsocketConn")
	}
}
