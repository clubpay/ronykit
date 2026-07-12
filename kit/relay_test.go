package kit

import (
	"errors"
	"net/http"
	"testing"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type relayHTTPCall struct {
	targetURL string
	cfg       RelayConfig
}

type testRelayConn struct {
	testRESTConn

	wsUpgrade      bool
	requestBody    []byte
	relayHTTPCalls []relayHTTPCall
	relayWSCalls   []relayHTTPCall
	writtenStatus  int
}

var _ RelayConn = (*testRelayConn)(nil)

func newTestRelayConn() *testRelayConn {
	return &testRelayConn{
		testRESTConn: testRESTConn{
			testConn: testConn{
				id: utils.RandomUint64(0),
				kv: map[string]string{},
			},
		},
	}
}

func (t *testRelayConn) RequestBody() ([]byte, error) {
	return t.requestBody, nil
}

func (t *testRelayConn) IsWebSocketUpgrade() bool {
	return t.wsUpgrade
}

func (t *testRelayConn) RelayHTTP(targetURL string, cfg RelayConfig) error {
	t.relayHTTPCalls = append(t.relayHTTPCalls, relayHTTPCall{targetURL: targetURL, cfg: cfg})

	return ErrRelayCompleted
}

func (t *testRelayConn) RelayWebSocket(targetURL string, cfg RelayConfig) error {
	t.relayWSCalls = append(t.relayWSCalls, relayHTTPCall{targetURL: targetURL, cfg: cfg})

	return ErrRelayCompleted
}

func (t *testRelayConn) WriteHTTPResponse(status int, _ http.Header, _ []byte) error {
	t.writtenStatus = status

	return ErrRelayCompleted
}

func TestRelayHTTP(t *testing.T) {
	t.Run("success stops execution", func(t *testing.T) {
		conn := newTestRelayConn()
		ctx := NewContext(nil)
		ctx.conn = conn
		ctx.in = newEnvelope(ctx, conn, false)

		var order []string
		ctx.handlers = []HandlerFunc{
			func(c *Context) {
				order = append(order, "h1")
				err := RelayHTTP(c, "http://upstream.example/api", RelayConfig{})
				require.NoError(t, err)
				order = append(order, "after-relay")
			},
			func(*Context) { order = append(order, "h2") },
		}
		ctx.Next()

		assert.Equal(t, []string{"h1", "after-relay"}, order)
		assert.Len(t, conn.relayHTTPCalls, 1)
		assert.Equal(t, "http://upstream.example/api", conn.relayHTTPCalls[0].targetURL)
		assert.Empty(t, conn.out)
	})

	t.Run("not supported", func(t *testing.T) {
		ctx := NewContext(nil)
		ctx.conn = newTestRESTConn()
		ctx.in = newEnvelope(ctx, ctx.conn, false)

		err := RelayHTTP(ctx, "http://upstream.example/api", RelayConfig{})
		require.ErrorIs(t, err, ErrRelayNotSupported)
	})
}

func TestRelayWebSocket(t *testing.T) {
	conn := newTestRelayConn()
	conn.wsUpgrade = true
	ctx := NewContext(nil)
	ctx.conn = conn
	ctx.in = newEnvelope(ctx, conn, false)

	err := RelayWebSocket(ctx, "ws://upstream.example/ws", RelayConfig{})
	require.NoError(t, err)
	assert.Len(t, conn.relayWSCalls, 1)
}

func TestRelayAutoDetect(t *testing.T) {
	t.Run("websocket", func(t *testing.T) {
		conn := newTestRelayConn()
		conn.wsUpgrade = true
		ctx := NewContext(nil)
		ctx.conn = conn
		ctx.in = newEnvelope(ctx, conn, false)

		err := Relay(ctx, "ws://upstream.example/ws", RelayConfig{})
		require.NoError(t, err)
		assert.Len(t, conn.relayWSCalls, 1)
		assert.Empty(t, conn.relayHTTPCalls)
	})

	t.Run("http", func(t *testing.T) {
		conn := newTestRelayConn()
		ctx := NewContext(nil)
		ctx.conn = conn
		ctx.in = newEnvelope(ctx, conn, false)

		err := Relay(ctx, "http://upstream.example/api", RelayConfig{})
		require.NoError(t, err)
		assert.Len(t, conn.relayHTTPCalls, 1)
	})
}

func TestInputBody(t *testing.T) {
	t.Run("relay conn", func(t *testing.T) {
		conn := newTestRelayConn()
		conn.requestBody = []byte("payload")
		ctx := NewContext(nil)
		ctx.conn = conn
		ctx.in = newEnvelope(ctx, conn, false)

		body, err := ctx.InputBody()
		require.NoError(t, err)
		assert.Equal(t, []byte("payload"), body)
	})

	t.Run("raw data fallback", func(t *testing.T) {
		ctx := NewContext(nil)
		ctx.conn = newTestRESTConn()
		ctx.in = newEnvelope(ctx, ctx.conn, false)
		ctx.rawData = []byte("raw")

		body, err := ctx.InputBody()
		require.NoError(t, err)
		assert.Equal(t, []byte("raw"), body)
	})
}

func TestRelayCompletedIsSuccessSentinel(t *testing.T) {
	err := ErrRelayCompleted
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRelayCompleted))
}

func TestWriteHTTPResponseViaRelayConn(t *testing.T) {
	conn := newTestRelayConn()
	err := conn.WriteHTTPResponse(http.StatusAccepted, http.Header{"X-Test": []string{"1"}}, []byte("ok"))
	require.ErrorIs(t, err, ErrRelayCompleted)
	assert.Equal(t, http.StatusAccepted, conn.writtenStatus)
}
