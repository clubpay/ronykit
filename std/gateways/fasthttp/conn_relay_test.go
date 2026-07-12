package fasthttp

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/fasthttp/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestRelayHTTP_forwardsMethodAndBody(t *testing.T) {
	const payload = `{"msg":"hi"}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/agent/v1/message:send", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.JSONEq(t, payload, string(body))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)

	ctx := newRequestCtx(MethodPost, "/relay")
	ctx.Request.SetBodyRaw([]byte(payload))
	ctx.Request.Header.SetContentType("application/json")

	conn := &httpConn{ctx: ctx}
	err := conn.RelayHTTP(upstream.URL+"/agent/v1/message:send", kit.RelayConfig{})
	require.ErrorIs(t, err, kit.ErrRelayCompleted)
	assert.JSONEq(t, `{"ok":true}`, string(ctx.Response.Body()))
}

func TestRelayHTTP_passthroughStatusAndHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"accepted"}`))
	}))
	t.Cleanup(upstream.Close)

	ctx := newRequestCtx(MethodGet, "/relay")
	conn := &httpConn{ctx: ctx}

	err := conn.RelayHTTP(upstream.URL+"/resource", kit.RelayConfig{})
	require.ErrorIs(t, err, kit.ErrRelayCompleted)
	assert.Equal(t, fasthttp.StatusAccepted, ctx.Response.StatusCode())
	assert.Equal(t, "application/problem+json", string(ctx.Response.Header.Peek("Content-Type")))
}

func TestRelayHTTP_dropsQueryParam(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("token"))
		assert.Equal(t, "1", r.URL.Query().Get("keep"))
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(upstream.Close)

	ctx := newRequestCtx(MethodGet, "/relay")
	conn := &httpConn{ctx: ctx}

	target := upstream.URL + "/resource?token=secret&keep=1"
	err := conn.RelayHTTP(target, kit.RelayConfig{DropQueryParams: []string{"token"}})
	require.ErrorIs(t, err, kit.ErrRelayCompleted)
}

func TestRelayHTTP_rewriteResponse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "http://internal.example/next")
		w.WriteHeader(http.StatusFound)
	}))
	t.Cleanup(upstream.Close)

	ctx := newRequestCtx(MethodGet, "/relay")
	conn := &httpConn{ctx: ctx}

	err := conn.RelayHTTP(upstream.URL+"/redirect", kit.RelayConfig{
		RewriteResponse: func(resp *kit.RelayResponseView) error {
			resp.Header.Set("Location", "https://public.example/next")

			return nil
		},
	})
	require.ErrorIs(t, err, kit.ErrRelayCompleted)
	assert.Equal(t, "https://public.example/next", string(ctx.Response.Header.Peek("Location")))
}

func TestRelayHTTP_rewriteRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/rewritten", r.URL.Path)
		assert.Equal(t, "yes", r.Header.Get("X-Rewritten"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, "new-body", string(body))

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	ctx := newRequestCtx(MethodPost, "/relay")
	ctx.Request.SetBodyRaw([]byte("orig-body"))
	conn := &httpConn{ctx: ctx}

	err := conn.RelayHTTP(upstream.URL+"/original", kit.RelayConfig{
		RewriteRequest: func(req *kit.RelayRequestView) error {
			req.URL.Path = "/rewritten"
			req.Header.Set("X-Rewritten", "yes")
			req.Body = []byte("new-body")

			return nil
		},
	})
	require.ErrorIs(t, err, kit.ErrRelayCompleted)
	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
}

func TestRelayWebSocket_rejectedOriginDoesNotComplete(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()

		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(upstream.Close)

	target := "ws" + strings.TrimPrefix(upstream.URL, "http") + "/ws"
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	errCh := make(chan error, 1)
	srv := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			conn := &httpConn{ctx: ctx}
			// Origin check rejects the upgrade; Upgrade returns an error before
			// Hijack, so the relay must close the dialed backend and not complete.
			errCh <- conn.RelayWebSocket(target, kit.RelayConfig{
				WebSocketCheckOrigin: func(string) bool { return false },
			})
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Shutdown() })

	_, _, err = websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/", nil)
	require.Error(t, err)

	select {
	case relayErr := <-errCh:
		require.Error(t, relayErr)
		assert.NotErrorIs(t, relayErr, kit.ErrRelayCompleted)
	case <-time.After(2 * time.Second):
		t.Fatal("relay handler did not return")
	}
}

func TestRelayWebSocket_echo(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer c.Close()

		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				return
			}

			require.NoError(t, c.WriteMessage(mt, msg))
		}
	}))
	t.Cleanup(upstream.Close)

	target := "ws" + strings.TrimPrefix(upstream.URL, "http") + "/ws"
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	srv := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			conn := &httpConn{ctx: ctx}
			err := conn.RelayWebSocket(target, kit.RelayConfig{})
			require.ErrorIs(t, err, kit.ErrRelayCompleted)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Shutdown() })

	client, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/", nil)
	require.NoError(t, err)
	defer client.Close()

	require.NoError(t, client.WriteMessage(websocket.TextMessage, []byte("ping")))
	_, msg, err := client.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, "ping", string(msg))

	require.NoError(t, client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")))
	time.Sleep(50 * time.Millisecond)
}

func TestRelayWebSocket_abruptCloseDoesNotPanic(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer c.Close()

		for {
			mt, msg, err := c.ReadMessage()
			if err != nil {
				return
			}

			require.NoError(t, c.WriteMessage(mt, msg))
		}
	}))
	t.Cleanup(upstream.Close)

	target := "ws" + strings.TrimPrefix(upstream.URL, "http")
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	srv := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			conn := &httpConn{ctx: ctx}
			err := conn.RelayWebSocket(target, kit.RelayConfig{})
			require.ErrorIs(t, err, kit.ErrRelayCompleted)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Shutdown() })

	for i := 0; i < 5; i++ {
		client, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/", nil)
		require.NoError(t, err)

		require.NoError(t, client.WriteMessage(websocket.TextMessage, []byte("ping")))
		_, msg, err := client.ReadMessage()
		require.NoError(t, err)
		assert.Equal(t, "ping", string(msg))

		require.NoError(t, client.UnderlyingConn().Close())
		time.Sleep(20 * time.Millisecond)
	}
}

func TestRelayWebSocket_subprotocols(t *testing.T) {
	var gotProtocol string
	upgrader := websocket.Upgrader{
		CheckOrigin:  func(_ *http.Request) bool { return true },
		Subprotocols: []string{"binary", "base64"},
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		gotProtocol = c.Subprotocol()
		defer c.Close()
		_ = c.WriteMessage(websocket.TextMessage, []byte("ok"))
	}))
	t.Cleanup(upstream.Close)

	target := "ws" + strings.TrimPrefix(upstream.URL, "http")
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	srv := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			conn := &httpConn{ctx: ctx}
			err := conn.RelayWebSocket(target, kit.RelayConfig{
				WebSocketSubprotocols: []string{"binary", "base64"},
			})
			require.ErrorIs(t, err, kit.ErrRelayCompleted)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Shutdown() })

	header := http.Header{}
	header.Set("Sec-WebSocket-Protocol", "base64")
	client, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/", header)
	require.NoError(t, err)
	defer client.Close()

	deadline := time.Now().Add(2 * time.Second)
	require.NoError(t, client.SetReadDeadline(deadline))
	_, msg, err := client.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, "ok", string(msg))
	assert.Equal(t, "base64", gotProtocol)

	require.NoError(t, client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")))
	time.Sleep(50 * time.Millisecond)
}

func TestRelayWebSocket_extraRequestHeaders(t *testing.T) {
	var gotHost string
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHost = r.Host
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		_ = c.WriteMessage(websocket.TextMessage, []byte("ok"))
	}))
	t.Cleanup(upstream.Close)

	upstreamHost := strings.TrimPrefix(upstream.URL, "http://")
	target := "ws://" + upstreamHost + "/ws"
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	srv := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			conn := &httpConn{ctx: ctx}
			err := conn.RelayWebSocket(target, kit.RelayConfig{
				ExtraRequestHeaders: map[string]string{"Host": upstreamHost},
			})
			require.ErrorIs(t, err, kit.ErrRelayCompleted)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Shutdown() })

	clientHeader := http.Header{}
	clientHeader.Set("Host", "localhost:8586")
	client, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/", clientHeader)
	require.NoError(t, err)
	defer client.Close()

	deadline := time.Now().Add(2 * time.Second)
	require.NoError(t, client.SetReadDeadline(deadline))
	_, _, err = client.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, upstreamHost, gotHost)
	assert.NotEqual(t, "localhost:8586", gotHost)

	require.NoError(t, client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")))
	time.Sleep(50 * time.Millisecond)
}

func TestRelayWebSocket_rewriteRequest(t *testing.T) {
	var gotHeader string
	upgrader := websocket.Upgrader{CheckOrigin: func(_ *http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Rewritten")
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		_ = c.WriteMessage(websocket.TextMessage, []byte("ok"))
	}))
	t.Cleanup(upstream.Close)

	target := "ws" + strings.TrimPrefix(upstream.URL, "http") + "/ws"
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	srv := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			conn := &httpConn{ctx: ctx}
			err := conn.RelayWebSocket(target, kit.RelayConfig{
				RewriteRequest: func(req *kit.RelayRequestView) error {
					req.Header.Set("X-Rewritten", "yes")

					return nil
				},
			})
			require.ErrorIs(t, err, kit.ErrRelayCompleted)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Shutdown() })

	client, _, err := websocket.DefaultDialer.Dial("ws://"+ln.Addr().String()+"/", nil)
	require.NoError(t, err)
	defer client.Close()

	deadline := time.Now().Add(2 * time.Second)
	require.NoError(t, client.SetReadDeadline(deadline))
	_, _, err = client.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, "yes", gotHeader)

	require.NoError(t, client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")))
	time.Sleep(50 * time.Millisecond)
}

func TestRelay_completedStopsExecution(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream"))
	}))
	t.Cleanup(upstream.Close)

	ctx := newRequestCtx(MethodGet, "/relay")
	conn := &httpConn{ctx: ctx}

	var sent bool
	err := kit.NewTestContext().
		Input(kit.RawMessage(nil), kit.EnvelopeHdr{}).
		SetHandler(
			func(c *kit.Context) {
				require.NoError(t, kit.RelayHTTP(c, upstream.URL, kit.RelayConfig{}))
			},
			func(c *kit.Context) {
				sent = true
				c.Out().SetMsg(kit.RawMessage("envelope")).Send()
			},
		).
		RunWithConn(conn)
	require.NoError(t, err)
	assert.False(t, sent)
	assert.Equal(t, "upstream", string(ctx.Response.Body()))
}

func TestRelay_notSupported(t *testing.T) {
	err := kit.NewTestContext().
		Input(kit.RawMessage(nil), kit.EnvelopeHdr{}).
		SetHandler(func(c *kit.Context) {
			require.ErrorIs(t, kit.RelayHTTP(c, "http://example.com", kit.RelayConfig{}), kit.ErrRelayNotSupported)
		}).
		RunWithConn(nonRelayRESTConn{})
	require.NoError(t, err)
}

type nonRelayRESTConn struct {
	id uint64
}

func (nonRelayRESTConn) ConnID() uint64                            { return 0 }
func (nonRelayRESTConn) ClientIP() string                          { return "" }
func (nonRelayRESTConn) WriteEnvelope(*kit.Envelope) error         { return nil }
func (nonRelayRESTConn) Stream() bool                              { return false }
func (nonRelayRESTConn) Walk(func(string, string) bool)            {}
func (nonRelayRESTConn) Get(string) string                         { return "" }
func (nonRelayRESTConn) Set(string, string)                        {}
func (nonRelayRESTConn) WalkQueryParams(func(string, string) bool) {}
func (nonRelayRESTConn) GetHost() string                           { return "" }
func (nonRelayRESTConn) GetRequestURI() string                     { return "" }
func (nonRelayRESTConn) GetPath() string                           { return "" }
func (nonRelayRESTConn) GetMethod() string                         { return http.MethodGet }
func (nonRelayRESTConn) SetStatusCode(int)                         {}
func (nonRelayRESTConn) Redirect(int, string)                      {}
