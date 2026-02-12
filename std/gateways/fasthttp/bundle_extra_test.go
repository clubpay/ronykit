package fasthttp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net"
	"testing"
	"testing/fstest"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/std/gateways/fasthttp/proxy"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

type testLogger struct {
	errors int
}

func (t *testLogger) Debugf(_ string, _ ...any) {}
func (t *testLogger) Errorf(_ string, _ ...any) { t.errors++ }

type captureDelegate struct {
	openCount  int
	closeCount int
	msgs       [][]byte
	msgCh      chan []byte
	closeCh    chan uint64
}

func (d *captureDelegate) OnOpen(_ kit.Conn) {
	d.openCount++
}

func (d *captureDelegate) OnClose(id uint64) {
	d.closeCount++
	if d.closeCh != nil {
		d.closeCh <- id
	}
}

func (d *captureDelegate) OnMessage(_ kit.Conn, msg []byte) {
	cp := append([]byte(nil), msg...)
	d.msgs = append(d.msgs, cp)
	if d.msgCh != nil {
		d.msgCh <- cp
	}
}

type wsReplyDelegate struct {
	msgCh    chan []byte
	closeCh  chan uint64
	envErrCh chan error
}

func (d *wsReplyDelegate) OnOpen(_ kit.Conn) {}

func (d *wsReplyDelegate) OnClose(id uint64) {
	if d.closeCh != nil {
		d.closeCh <- id
	}
}

func (d *wsReplyDelegate) OnMessage(c kit.Conn, msg []byte) {
	if d.msgCh != nil {
		d.msgCh <- append([]byte(nil), msg...)
	}
	if w, ok := c.(kit.RPCConn); ok {
		_, _ = w.Write([]byte("pong"))
	}
	env := newTestEnvelope(newTestContext(c), c)
	env.SetID("1").SetHdr("X-Env", "1").SetMsg(kit.RawMessage(`"env"`))
	err := c.WriteEnvelope(env)
	if d.envErrCh != nil {
		d.envErrCh <- err
	}
}

func TestNewWithOptions(t *testing.T) {
	logger := &testLogger{}
	fs := fstest.MapFS{
		"index.html": {Data: []byte("ok")},
	}

	gw, err := New(
		SuperFast(),
		WithServerName("unit-test"),
		WithBufferSize(128, 256),
		WithLogger(logger),
		Listen("127.0.0.1:0"),
		WithCORS(CORSConfig{AllowedOrigins: []string{"*"}}),
		WithPredicateKey("pred"),
		WithWebsocketEndpoint("/ws"),
		WithReverseProxy("", proxy.WithAddress("127.0.0.1:1")),
		WithCustomRPC(common.SimpleIncomingJSONRPC, common.SimpleOutgoingJSONRPC),
		WithDisableHeaderNamesNormalizing(),
		WithServeFS("/static", ".", fs),
		WithMaxRequestBodySize(32<<10),
		WithCompressionLevel(CompressionLevelBestSpeed),
		WithAutoDecompressRequests(true),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	b := gw.(*bundle) //nolint:forcetypeassert
	if b.listen != "127.0.0.1:0" {
		t.Fatalf("unexpected listen address: %s", b.listen)
	}
	if b.srv.Name != "unit-test" {
		t.Fatalf("unexpected server name: %s", b.srv.Name)
	}
	if b.srv.ReadBufferSize != 128 || b.srv.WriteBufferSize != 256 {
		t.Fatalf("unexpected buffer sizes: %d/%d", b.srv.ReadBufferSize, b.srv.WriteBufferSize)
	}
	if b.cors == nil || b.wsEndpoint != "/ws" {
		t.Fatalf("expected cors and websocket endpoint to be set")
	}
	if b.reverseProxy == nil || b.reverseProxyPath != "/" {
		t.Fatalf("expected reverse proxy to be set")
	}
	if b.predicateKey != "pred" {
		t.Fatalf("unexpected predicate key: %s", b.predicateKey)
	}
	if b.srv.MaxRequestBodySize != 32<<10 {
		t.Fatalf("unexpected max request size: %d", b.srv.MaxRequestBodySize)
	}
	if b.compress != CompressionLevelBestSpeed || !b.autoDecompress {
		t.Fatalf("unexpected compression settings")
	}
	if b.srv.DisableHeaderNamesNormalizing != true {
		t.Fatalf("expected header normalizing to be disabled")
	}
	if logger.errors != 0 {
		t.Fatalf("unexpected logger error count: %d", logger.errors)
	}

	_ = MustNew(WithCompressionLevel(CompressionLevelBestCompression))
}

func TestRegisterRPCAndREST(t *testing.T) {
	gw, _ := New(WithCORS(CORSConfig{}))
	b := gw.(*bundle) //nolint:forcetypeassert
	delegate := &captureDelegate{msgCh: make(chan []byte, 1)}
	b.Subscribe(delegate)

	b.Register("svc", "c1", kit.JSON, RPC("pred"), kit.RawMessage{}, kit.RawMessage{})
	if b.wsRoutes["pred"] == nil {
		t.Fatalf("expected ws route to be registered")
	}

	b.registerRPC("svc", "c1", kit.JSON, Selector{}, kit.RawMessage{})
	if len(b.wsRoutes) != 1 {
		t.Fatalf("unexpected ws route count: %d", len(b.wsRoutes))
	}

	customDec := func(_ *RequestCtx, data []byte) (kit.Message, error) {
		return kit.RawMessage(data), nil
	}
	restSel := REST(MethodPost, "/echo").SetDecoder(customDec)
	b.Register("svc", "c2", kit.JSON, restSel, kit.RawMessage{}, kit.RawMessage{})

	ctx := newRequestCtx(MethodPost, "/echo")
	ctx.Request.SetBodyRaw([]byte("body"))
	b.httpRouter.Handler(ctx)

	select {
	case got := <-delegate.msgCh:
		if string(got) != "body" {
			t.Fatalf("unexpected message: %s", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive message")
	}
}

func TestGenHTTPHandlerAutoDecompress(t *testing.T) {
	logger := &testLogger{}
	gw, _ := New(WithLogger(logger), WithAutoDecompressRequests(true))
	b := gw.(*bundle) //nolint:forcetypeassert
	delegate := &captureDelegate{msgCh: make(chan []byte, 1)}
	b.Subscribe(delegate)

	handler := b.genHTTPHandler(routeData{})
	ctx := newRequestCtx(MethodPost, "/")
	body := []byte("hello")
	compressed := fasthttp.AppendGzipBytes(nil, body)
	ctx.Request.Header.SetContentEncoding("gzip")
	ctx.Request.SetBodyRaw(compressed)

	handler(ctx)
	select {
	case got := <-delegate.msgCh:
		if string(got) != "hello" {
			t.Fatalf("unexpected decompressed body: %s", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive decompressed message")
	}

	ctx2 := newRequestCtx(MethodPost, "/")
	ctx2.Request.Header.SetContentEncoding("br")
	ctx2.Request.SetBodyRaw([]byte("bad"))
	handler(ctx2)
	if logger.errors == 0 {
		t.Fatalf("expected decompression error to be logged")
	}
}

func TestHTTPDispatchPaths(t *testing.T) {
	gw, _ := New()
	b := gw.(*bundle) //nolint:forcetypeassert

	conn := &httpConn{ctx: newRequestCtx(MethodOptions, "/")}
	ctx := newTestContext(conn)
	_, err := b.httpDispatch(ctx, nil)
	if !errors.Is(err, kit.ErrPreflight) {
		t.Fatalf("expected preflight error, got %v", err)
	}

	conn.ctx = newRequestCtx(MethodGet, "/")
	_, err = b.httpDispatch(ctx, nil)
	if !errors.Is(err, kit.ErrNoHandler) {
		t.Fatalf("expected no handler error, got %v", err)
	}

	conn.rd = &routeData{
		Decoder: func(_ *RequestCtx, _ []byte) (kit.Message, error) {
			return nil, errors.New("decode failed")
		},
	}
	_, err = b.httpDispatch(ctx, []byte("x"))
	if !errors.Is(err, kit.ErrDecodeIncomingMessageFailed) {
		t.Fatalf("expected decode error, got %v", err)
	}

	conn.rd = &routeData{
		Method:      MethodPost,
		Path:        "/",
		ServiceName: "svc",
		ContractID:  "c1",
		Decoder: func(_ *RequestCtx, data []byte) (kit.Message, error) {
			return kit.RawMessage(data), nil
		},
	}
	conn.ctx.Request.Header.Set("X-Test", "ok")
	arg, err := b.httpDispatch(ctx, []byte("payload"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if arg.ServiceName != "svc" || arg.ContractID != "c1" {
		t.Fatalf("unexpected execute arg: %+v", arg)
	}
	if got := ctx.In().GetHdr("X-Test"); got != "ok" {
		t.Fatalf("expected header to be walked, got %q", got)
	}
}

type wsPayload struct {
	Name string `json:"name"`
}

type incomingEnvelope struct {
	ID      string            `json:"id"`
	Header  map[string]string `json:"hdr"`
	Payload json.RawMessage   `json:"payload"`
}

type incomingEnvelopeB64 struct {
	ID      string            `json:"id"`
	Header  map[string]string `json:"hdr"`
	Payload string            `json:"payload"`
}

func TestWSDispatchPaths(t *testing.T) {
	gw, _ := New(WithPredicateKey("pred"))
	b := gw.(*bundle) //nolint:forcetypeassert

	b.wsRoutes["route"] = &routeData{
		Predicate:   "route",
		ServiceName: "svc",
		ContractID:  "c1",
		Factory:     kit.CreateMessageFactory(&wsPayload{}),
	}
	b.wsRoutes["raw"] = &routeData{
		Predicate:   "raw",
		ServiceName: "svc",
		ContractID:  "c2",
		Factory:     kit.CreateMessageFactory(kit.RawMessage{}),
	}
	b.wsRoutes["form"] = &routeData{
		Predicate:   "form",
		ServiceName: "svc",
		ContractID:  "c3",
		Factory:     kit.CreateMessageFactory(&kit.MultipartFormMessage{}),
	}

	ctx := newTestContext(&wsConn{kv: map[string]string{}})
	_, err := b.wsDispatch(ctx, nil)
	if !errors.Is(err, kit.ErrDecodeIncomingContainerFailed) {
		t.Fatalf("expected decode container error, got %v", err)
	}

	missing, _ := json.Marshal(incomingEnvelope{ID: "1", Header: map[string]string{"pred": "missing"}})
	_, err = b.wsDispatch(ctx, missing)
	if !errors.Is(err, kit.ErrNoHandler) {
		t.Fatalf("expected no handler error, got %v", err)
	}

	payload, _ := json.Marshal(wsPayload{Name: "alice"})
	enc, _ := json.Marshal(incomingEnvelope{
		ID:      "2",
		Header:  map[string]string{"pred": "route"},
		Payload: payload,
	})
	arg, err := b.wsDispatch(ctx, enc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg := ctx.In().GetMsg().(*wsPayload) //nolint:forcetypeassert
	if msg.Name != "alice" || arg.Route != "route" {
		t.Fatalf("unexpected ws dispatch result")
	}

	rawBody := []byte("raw-data")
	encRaw, _ := json.Marshal(incomingEnvelopeB64{
		ID:      "3",
		Header:  map[string]string{"pred": "raw"},
		Payload: base64.StdEncoding.EncodeToString(rawBody),
	})
	_, err = b.wsDispatch(ctx, encRaw)
	if err != nil {
		t.Fatalf("unexpected raw error: %v", err)
	}
	expectedRaw := `"` + base64.StdEncoding.EncodeToString(rawBody) + `"`
	if string(ctx.In().GetMsg().(kit.RawMessage)) != expectedRaw { //nolint:forcetypeassert
		t.Fatalf("unexpected raw message: %s", ctx.In().GetMsg())
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("a", "b")
	_ = writer.Close()
	encForm, _ := json.Marshal(incomingEnvelopeB64{
		ID: "4",
		Header: map[string]string{
			"pred":         "form",
			"Content-Type": writer.FormDataContentType(),
		},
		Payload: base64.StdEncoding.EncodeToString(body.Bytes()),
	})
	_, err = b.wsDispatch(ctx, encForm)
	if !errors.Is(err, kit.ErrDecodeIncomingMessageFailed) {
		t.Fatalf("expected form decode error, got %v", err)
	}
}

func TestDispatchUnknownConn(t *testing.T) {
	gw, _ := New()
	b := gw.(*bundle) //nolint:forcetypeassert

	ctx := newTestContext(&struct{ kit.Conn }{})
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic for unknown connection")
		}
	}()
	_, _ = b.Dispatch(ctx, []byte("x"))
}

func TestWSHandler(t *testing.T) {
	gw, _ := New(WithWebsocketEndpoint("/ws"))
	b := gw.(*bundle) //nolint:forcetypeassert

	msgCh := make(chan []byte, 1)
	closeCh := make(chan uint64, 1)
	envErrCh := make(chan error, 1)
	delegate := &wsReplyDelegate{msgCh: msgCh, closeCh: closeCh, envErrCh: envErrCh}
	b.Subscribe(delegate)

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	go func() {
		_ = b.srv.Serve(ln)
	}()
	defer b.srv.Shutdown()

	wsURL := "ws://" + ln.Addr().String() + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial failed: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	select {
	case got := <-msgCh:
		if string(got) != "ping" {
			t.Fatalf("unexpected ws message: %s", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive ws message")
	}

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, pong, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(pong) != "pong" {
		t.Fatalf("unexpected pong: %s", pong)
	}

	select {
	case err := <-envErrCh:
		if err != nil {
			t.Fatalf("write envelope failed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive envelope write signal")
	}

	_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"), time.Now().Add(time.Second))
	select {
	case <-closeCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("did not receive close")
	}
}
