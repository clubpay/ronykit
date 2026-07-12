package rony

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clubpay/ronykit/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupWithRelay(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("relayed"))
	}))
	t.Cleanup(upstream.Close)

	srv := NewServer(Listen(":0"))
	Setup[EMPTY, NOP](
		srv,
		"deploy",
		EmptyState(),
		WithRelay(func(ctx *SRelayCtx) error {
			return ctx.Relay(upstream.URL, kit.RelayConfig{})
		},
			RelayALL("/deploy/v1/sessions/{sessionId}/api/{path:*}"),
		),
	)

	svc := srv.cfg.services["deploy"]
	require.NotNil(t, svc)
	require.Len(t, svc.Contracts, 1)

	contract := svc.Contracts[0]
	require.NotEmpty(t, contract.Handlers)

	conn := &relayIntegrationConn{t: t}
	err := kit.NewTestContext().
		Input(kit.RawMessage(nil), kit.EnvelopeHdr{}).
		SetHandler(contract.Handlers...).
		RunWithConn(conn)
	require.NoError(t, err)
	assert.Equal(t, "relayed", string(conn.responseBody))
	assert.False(t, conn.envelopeSent)
}

func TestRelayCtxIsWebSocketUpgrade(t *testing.T) {
	conn := &relayIntegrationConn{t: t, wsUpgrade: true}
	err := kit.NewTestContext().
		Input(kit.RawMessage(nil), kit.EnvelopeHdr{}).
		SetHandler(func(c *kit.Context) {
			relayCtx := newRelayCtx[EMPTY, NOP](c, ptr(EMPTY{}), nil)
			assert.True(t, relayCtx.IsWebSocketUpgrade())
		}).
		RunWithConn(conn)
	require.NoError(t, err)
}

type relayIntegrationConn struct {
	t            *testing.T
	wsUpgrade    bool
	responseBody []byte
	envelopeSent bool
}

func (c *relayIntegrationConn) ConnID() uint64                 { return 1 }
func (c *relayIntegrationConn) ClientIP() string               { return "127.0.0.1" }
func (c *relayIntegrationConn) Stream() bool                   { return false }
func (c *relayIntegrationConn) Walk(func(string, string) bool) {}
func (c *relayIntegrationConn) Get(string) string              { return "" }
func (c *relayIntegrationConn) Set(string, string)             {}

func (c *relayIntegrationConn) WriteEnvelope(*kit.Envelope) error {
	c.envelopeSent = true
	return nil
}

func (c *relayIntegrationConn) GetMethod() string                         { return http.MethodGet }
func (c *relayIntegrationConn) GetHost() string                           { return "example.com" }
func (c *relayIntegrationConn) GetRequestURI() string                     { return "/deploy/v1/sessions/s1/api/x" }
func (c *relayIntegrationConn) GetPath() string                           { return "/deploy/v1/sessions/s1/api/x" }
func (c *relayIntegrationConn) SetStatusCode(int)                         {}
func (c *relayIntegrationConn) Redirect(int, string)                      {}
func (c *relayIntegrationConn) WalkQueryParams(func(string, string) bool) {}

func (c *relayIntegrationConn) RequestBody() ([]byte, error) { return nil, nil }
func (c *relayIntegrationConn) IsWebSocketUpgrade() bool     { return c.wsUpgrade }

func (c *relayIntegrationConn) RelayHTTP(targetURL string, _ kit.RelayConfig) error {
	resp, err := http.Get(targetURL)
	require.NoError(c.t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err)
	c.responseBody = body

	return kit.ErrRelayCompleted
}

func (c *relayIntegrationConn) RelayWebSocket(string, kit.RelayConfig) error {
	return kit.ErrRelayCompleted
}

func (c *relayIntegrationConn) WriteHTTPResponse(int, http.Header, []byte) error {
	return kit.ErrRelayCompleted
}

func ptr[T any](v T) *T { return &v }
