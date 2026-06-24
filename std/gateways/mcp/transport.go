package mcp

// Transport selects the MCP wire protocol used by the gateway.
type Transport string

const (
	// TransportStreamableHTTP uses the MCP Streamable HTTP transport (default).
	TransportStreamableHTTP Transport = "streamable-http"
	// TransportSSE uses the legacy MCP HTTP+SSE transport (2024-11-05 spec).
	TransportSSE Transport = "sse"
	// TransportStdio uses newline-delimited JSON over stdin/stdout.
	TransportStdio Transport = "stdio"
)
