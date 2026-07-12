package kit

import (
	"io"
	"net/http"
)

// Conn represents a connection between EdgeServer and client.
type Conn interface {
	ConnID() uint64
	ClientIP() string
	WriteEnvelope(e *Envelope) error
	Stream() bool
	Walk(fn func(key string, val string) bool)
	Get(key string) string
	Set(key string, val string)
}

// RESTConn implemented by Gateway, so in Dispatcher user can check if Conn also implements
// RESTConn, then it has more information about the RESTConn request.
type RESTConn interface {
	Conn
	GetMethod() string
	GetHost() string
	// GetRequestURI returns uri without Method and Host
	GetRequestURI() string
	// GetPath returns uri without Method, Host, and Query parameters.
	GetPath() string
	SetStatusCode(code int)
	Redirect(code int, url string)
	WalkQueryParams(fn func(key string, val string) bool)
}

type RPCConn interface {
	Conn
	io.Writer
	Close()
}

// RelayConn is implemented by gateway connections that support
// handler-initiated HTTP and WebSocket reverse proxying.
type RelayConn interface {
	RESTConn

	// RequestBody returns the inbound request body (decompressed when supported).
	RequestBody() ([]byte, error)

	// IsWebSocketUpgrade reports whether the inbound request is a WS upgrade.
	IsWebSocketUpgrade() bool

	// RelayHTTP forwards the current inbound HTTP request to targetURL and writes
	// the upstream response to the client. targetURL must be absolute (scheme + host + path + query).
	// The inbound HTTP method is preserved. Returns ErrRelayCompleted on success.
	RelayHTTP(targetURL string, cfg RelayConfig) error

	// RelayWebSocket upgrades the client connection and proxies frames to/from targetURL.
	// targetURL must use ws:// or wss://. Returns ErrRelayCompleted on success.
	RelayWebSocket(targetURL string, cfg RelayConfig) error

	// WriteHTTPResponse writes a pre-built response (status, headers, body) directly to
	// the client without envelope encoding. For advanced callers only.
	WriteHTTPResponse(status int, header http.Header, body []byte) error
}
