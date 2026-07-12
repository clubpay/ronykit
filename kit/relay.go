package kit

import (
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"time"
)

// RelayConfig controls relay behavior. Zero value = sensible defaults.
//
// Handler relay forwards the inbound request to a dynamic targetURL after application
// logic (auth, session lookup). Use from kit.Relay or rony.RelayCtx.Relay on routes
// registered with rony.WithRelay. For static gateway-level proxying use
// rony.WithReverseProxy instead.
type RelayConfig struct {
	ExtraRequestHeaders map[string]string
	DropRequestHeaders  []string // merged with default hop-by-hop list
	DropQueryParams     []string

	TLSConfig *tls.Config
	Timeout   time.Duration // 0 = no timeout

	// RewriteRequest is called after building the outbound request, before send.
	RewriteRequest func(req *RelayRequestView) error

	// RewriteResponse is called on the upstream response before writing to client.
	RewriteResponse func(resp *RelayResponseView) error

	// WebSocket-only
	WebSocketSubprotocols []string
	WebSocketCheckOrigin  func(origin string) bool
}

type RelayRequestView struct {
	Method string
	URL    *url.URL
	Header http.Header
	Body   []byte
}

type RelayResponseView struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

var (
	ErrRelayNotSupported = errors.New("relay not supported by connection")
	ErrRelayCompleted    = errors.New("relay completed") // success sentinel; stop execution
)

func (ctx *Context) RelayConn() (RelayConn, bool) {
	rc, ok := ctx.conn.(RelayConn)

	return rc, ok
}

func (ctx *Context) IsRelay() bool {
	_, ok := ctx.conn.(RelayConn)

	return ok
}

// InputBody returns the inbound request body. When the connection implements RelayConn,
// RequestBody is used (including decompression when Content-Encoding is set).
// Otherwise InputRawData is returned when present.
func (ctx *Context) InputBody() ([]byte, error) {
	if rc, ok := ctx.conn.(RelayConn); ok {
		return rc.RequestBody()
	}

	if len(ctx.rawData) > 0 {
		return ctx.rawData, nil
	}

	return nil, nil
}

// RelayHTTP relays via RelayConn and calls StopExecution on success.
func RelayHTTP(ctx *Context, targetURL string, cfg RelayConfig) error {
	rc, ok := ctx.RelayConn()
	if !ok {
		return ErrRelayNotSupported
	}

	err := rc.RelayHTTP(targetURL, cfg)
	if errors.Is(err, ErrRelayCompleted) {
		ctx.StopExecution()

		return nil
	}

	return err
}

// RelayWebSocket relays via RelayConn and calls StopExecution on success.
func RelayWebSocket(ctx *Context, targetURL string, cfg RelayConfig) error {
	rc, ok := ctx.RelayConn()
	if !ok {
		return ErrRelayNotSupported
	}

	err := rc.RelayWebSocket(targetURL, cfg)
	if errors.Is(err, ErrRelayCompleted) {
		ctx.StopExecution()

		return nil
	}

	return err
}

// Relay forwards HTTP or WebSocket based on the Upgrade header.
//
// Returns ErrRelayNotSupported when the connection does not implement RelayConn.
// On success (ErrRelayCompleted from RelayConn), StopExecution is called and nil is returned.
func Relay(ctx *Context, targetURL string, cfg RelayConfig) error {
	rc, ok := ctx.RelayConn()
	if !ok {
		return ErrRelayNotSupported
	}

	if rc.IsWebSocketUpgrade() {
		return RelayWebSocket(ctx, targetURL, cfg)
	}

	return RelayHTTP(ctx, targetURL, cfg)
}
