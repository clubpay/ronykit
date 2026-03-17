package mcp

import (
	"net"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Option = func(*bundle)

func WithName(name string) Option {
	return func(b *bundle) {
		b.name = name
	}
}

func WithTitle(title string) Option {
	return func(b *bundle) {
		b.title = title
	}
}

func WithWebsiteURL(url string) Option {
	return func(b *bundle) {
		b.websiteURL = url
	}
}

func WithInstructions(instructions string) Option {
	return func(b *bundle) {
		b.instructions = instructions
	}
}

// WithAddr sets the TCP listen address for the Streamable HTTP server.
// Default is ":8080".
func WithAddr(addr string) Option {
	return func(b *bundle) {
		b.addr = addr
	}
}

// WithListener provides a pre-bound listener (useful for tests and reuseport setups).
// If set, WithAddr is ignored.
func WithListener(ln net.Listener) Option {
	return func(b *bundle) {
		b.ln = ln
	}
}

// WithHTTPServer allows customizing the underlying http.Server.
// Fields like Handler will be overwritten by the gateway at Start().
func WithHTTPServer(srv *http.Server) Option {
	return func(b *bundle) {
		b.httpSrv = srv
	}
}

// WithStreamableHTTPOptions customizes the MCP Streamable HTTP handler behavior.
func WithStreamableHTTPOptions(opts mcp.StreamableHTTPOptions) Option {
	return func(b *bundle) {
		b.streamableOpts = opts
	}
}

// WithEventStore enables SSE replay/resumption for Streamable HTTP.
func WithEventStore(store mcp.EventStore) Option {
	return func(b *bundle) {
		b.streamableOpts.EventStore = store
	}
}

// WithSessionTimeout configures idle session cleanup for Streamable HTTP sessions.
func WithSessionTimeout(d time.Duration) Option {
	return func(b *bundle) {
		b.streamableOpts.SessionTimeout = d
	}
}

// WithJSONResponse forces application/json responses for POST requests (no SSE response body).
func WithJSONResponse(enabled bool) Option {
	return func(b *bundle) {
		b.streamableOpts.JSONResponse = enabled
	}
}

// WithStateless enables streamable HTTP stateless mode (no persistent sessions).
func WithStateless(enabled bool) Option {
	return func(b *bundle) {
		b.streamableOpts.Stateless = enabled
	}
}

// WithServerOptions customizes MCP server behavior (capabilities, logging, handlers, keepalive, etc).
func WithServerOptions(opts mcp.ServerOptions) Option {
	return func(b *bundle) {
		b.serverOpts = opts
	}
}

// WithServerConfig allows last-mile mutations on the constructed MCP server.
func WithServerConfig(fn func(*mcp.Server)) Option {
	return func(b *bundle) {
		b.serverConfigFns = append(b.serverConfigFns, fn)
	}
}
