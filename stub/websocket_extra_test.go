package stub

import (
	"context"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/fasthttp/websocket"
)

type wsPayload struct {
	Text string `json:"text"`
}

func TestWebsocketDoAndStats(t *testing.T) {
	host, stop := startWebsocketServer(t)
	defer stop()

	wCtx := New(host).Websocket(
		WithPredicateKey("cmd"),
		WithAutoReconnect(false),
		WithPingTime(time.Second),
	)

	if err := wCtx.Connect(context.Background(), "ws"); err != nil {
		t.Fatal(err)
	}

	defer wCtx.Disconnect()

	var (
		wg     sync.WaitGroup
		gotErr error
		gotHdr Header
		gotMsg wsPayload
	)
	gotErr = nil
	wg.Add(1)

	err := wCtx.TextMessage(
		context.Background(),
		"echo",
		&wsPayload{Text: "ping"},
		&gotMsg,
		func(_ context.Context, msg kit.Message, hdr Header, err error) {
			defer wg.Done()
			gotErr = err
			gotHdr = hdr
			gotMsg = *msg.(*wsPayload) //nolint:forcetypeassert
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	wg.Wait()

	if gotErr != nil {
		t.Fatalf("unexpected callback error: %v", gotErr)
	}
	if gotHdr["cmd"] != "echo" {
		t.Fatalf("unexpected header: %v", gotHdr)
	}
	if gotMsg.Text != "ping-reply" {
		t.Fatalf("unexpected response: %+v", gotMsg)
	}

	stats := wCtx.Stats()
	if stats.WriteBytes == 0 || stats.ReadBytes == 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	if wCtx.NetConn() == nil {
		t.Fatal("expected net conn")
	}

	wg.Add(1)
	err = wCtx.BinaryMessage(
		context.Background(),
		"echo",
		&wsPayload{Text: "bin"},
		&gotMsg,
		func(_ context.Context, msg kit.Message, _ Header, err error) {
			defer wg.Done()
			if err != nil {
				gotErr = err
				return
			}
			gotMsg = *msg.(*wsPayload) //nolint:forcetypeassert
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if gotErr != nil || gotMsg.Text != "bin-reply" {
		t.Fatalf("unexpected binary response: %v %+v", gotErr, gotMsg)
	}
}

func TestWebsocketTimeoutAndRecover(t *testing.T) {
	host, stop := startWebsocketServer(t)
	defer stop()

	panicHit := make(chan any, 1)

	wCtx := New(host).Websocket(
		WithPredicateKey("cmd"),
		WithAutoReconnect(false),
		WithRecoverPanic(func(err any) {
			panicHit <- err
		}),
	)

	if err := wCtx.Connect(context.Background(), "ws"); err != nil {
		t.Fatal(err)
	}
	defer wCtx.Disconnect()

	if err := wCtx.Reconnect(context.Background()); err != nil {
		t.Fatalf("unexpected reconnect error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	done := make(chan error, 1)
	err := wCtx.TextMessage(
		ctx,
		"no-reply",
		&wsPayload{Text: "timeout"},
		&wsPayload{},
		func(_ context.Context, _ kit.Message, _ Header, err error) {
			if err != ErrTimeout {
				done <- err
				return
			}
			done <- nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected callback error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for callback")
	}

	func() {
		defer wCtx.recoverPanic()
		panic("boom")
	}()

	select {
	case <-panicHit:
	case <-time.After(time.Second):
		t.Fatal("expected panic recovery")
	}

	wCtx.setActivity()
	if wCtx.getActivity() == 0 {
		t.Fatal("expected activity timestamp")
	}
}

func TestWebsocketOptions(t *testing.T) {
	cfg := wsConfig{
		rateLimitChan: make(chan struct{}, 1),
	}
	WithUpgradeHeader("X-Test", "1")(&cfg)
	customDialer := func() *websocket.Dialer { return &websocket.Dialer{} }
	WithCustomDialerBuilder(customDialer)(&cfg)
	WithDefaultHandler(func(_ context.Context, _ kit.IncomingRPCContainer) {})(&cfg)
	WithHandler("echo", func(_ context.Context, _ kit.IncomingRPCContainer) {})(&cfg)
	WithCustomRPC(common.SimpleIncomingJSONRPC, common.SimpleOutgoingJSONRPC)(&cfg)
	WithOnConnectHandler(func(_ *WebsocketCtx) {})(&cfg)
	WithPreDialHandler(func(_ *Dialer) {})(&cfg)
	WithPredicateKey("cmd")(&cfg)
	WithConcurrency(10)(&cfg)
	WithAutoReconnect(true)(&cfg)
	WithPingTime(time.Second)(&cfg)
	WithCompression(CompressionBestSpeed)(&cfg)
	WithPreflightRPC(func(req *WebsocketRequest) { req.ReqHdr = Header{"x": "y"} })(&cfg)

	if cfg.upgradeHdr.Get("X-Test") != "1" || cfg.predicateKey != "cmd" {
		t.Fatalf("unexpected upgrade header or predicate key: %v", cfg.upgradeHdr)
	}
	if cfg.dialerBuilder == nil || cfg.defaultHandler == nil || cfg.handlers["echo"] == nil {
		t.Fatalf("expected custom handlers and dialer builder")
	}
	if cap(cfg.rateLimitChan) != 10 || !cfg.autoReconnect {
		t.Fatalf("unexpected concurrency or reconnect values")
	}
	if cfg.compressLevel == 0 || cfg.pingTime != time.Second {
		t.Fatalf("unexpected compression or ping time")
	}
	if len(cfg.preflights) != 1 {
		t.Fatalf("expected preflight")
	}
	if cfg.rpcInFactory == nil || cfg.rpcOutFactory == nil {
		t.Fatalf("expected custom rpc factories")
	}
	if cfg.preDial == nil || cfg.onConnect == nil {
		t.Fatalf("expected pre-dial and on-connect")
	}
}

func TestContainerTraceCarrier(t *testing.T) {
	out := &traceOut{}
	in := &traceIn{hdr: map[string]string{"trace": "1"}}
	c := containerTraceCarrier{out: out, in: in}
	if c.Get("trace") != "1" {
		t.Fatalf("unexpected trace get")
	}
	c.Set("trace", "2")
	if out.vals["trace"] != "2" {
		t.Fatalf("unexpected trace set: %v", out.vals)
	}
}

func startWebsocketServer(t *testing.T) (string, func()) {
	t.Helper()

	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(_ *http.Request) bool { return true },
	}

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}

			go func() {
				defer func() {
					_ = conn.Close()
				}()

				for {
					mt, msg, err := conn.ReadMessage()
					if err != nil {
						return
					}

					in := common.SimpleIncomingJSONRPC()
					if err := in.Unmarshal(msg); err != nil {
						in.Release()
						continue
					}

					predicate := in.GetHdr("cmd")
					if predicate == "no-reply" {
						in.Release()
						continue
					}

					var req wsPayload
					_ = in.ExtractMessage(&req)

					out := common.SimpleOutgoingJSONRPC()
					out.SetID(in.GetID())
					out.SetHdr("cmd", predicate)
					out.InjectMessage(&wsPayload{Text: req.Text + "-reply"})
					data, _ := out.Marshal()
					_ = conn.WriteMessage(mt, data)

					in.Release()
					out.Release()
				}
			}()
		}),
	}

	go func() {
		_ = server.Serve(ln)
	}()

	return ln.Addr().String(), func() {
		_ = server.Close()
		_ = ln.Close()
	}
}

type traceOut struct {
	vals map[string]string
}

func (t *traceOut) SetID(_ string) {}
func (t *traceOut) Marshal() ([]byte, error) {
	return nil, nil
}
func (t *traceOut) SetHdr(k, v string) {
	if t.vals == nil {
		t.vals = map[string]string{}
	}
	t.vals[k] = v
}
func (t *traceOut) InjectMessage(_ kit.Message) {}
func (t *traceOut) Release()                    {}

type traceIn struct {
	hdr map[string]string
}

func (t *traceIn) GetID() string                      { return "" }
func (t *traceIn) Unmarshal(_ []byte) error           { return nil }
func (t *traceIn) ExtractMessage(_ kit.Message) error { return nil }
func (t *traceIn) GetHdr(key string) string           { return t.hdr[key] }
func (t *traceIn) GetHdrMap() map[string]string       { return t.hdr }
func (t *traceIn) Release()                           {}
