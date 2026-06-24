package intent

import (
	"context"
	"encoding/json"
)

// Adapter notes for github.com/modelcontextprotocol/go-sdk/mcp:
//
//   - MCPServer            -> mcp.ClientSession (via mcp.Client.Connect)
//   - MCPServerConfig      -> mcp.StreamableClientTransport, mcp.SSEClientTransport, mcp.StdioTransport
//   - MCPTool              -> mcp.Tool
//   - MCPToolResult        -> mcp.CallToolResult
//   - MCPContentBlock      -> mcp.Content implementations
//   - MCPResourceSummary   -> mcp.Resource
//   - MCPPromptSummary     -> mcp.Prompt
//   - MCPPromptResult      -> mcp.GetPromptResult
//   - MCPClientFactory     -> mcp.NewClient + transport selection
//
// std/gateways/mcp hosts MCP servers (agent exposes tools). This package models
// the agent as an MCP client connecting to external servers.

// MCPTransportKind selects how to connect to an MCP server.
// Adapters map to github.com/modelcontextprotocol/go-sdk/mcp transports.
type MCPTransportKind string

const (
	MCPTransportStreamableHTTP MCPTransportKind = "streamable-http"
	MCPTransportSSE            MCPTransportKind = "sse"
	MCPTransportStdio          MCPTransportKind = "stdio"
)

// MCPServerConfig describes how to connect to an external MCP server.
type MCPServerConfig struct {
	Name      string
	Transport MCPTransportKind
	URL       string   // HTTP/SSE/streamable HTTP endpoint
	Command   []string // stdio: command and args
	Meta      map[string]string
}

// MCPTool is a tool exposed by an MCP server.
// Adapters map from mcp.Tool.
type MCPTool struct {
	Name         string
	Title        string
	Description  string
	InputSchema  json.RawMessage
	OutputSchema json.RawMessage
}

// MCPContentBlock is an unstructured tool output.
// Adapters map from mcp.Content (TextContent, ImageContent, etc.).
type MCPContentBlock struct {
	MIMEType string
	Text     string
	Binary   []byte
	URI      string
}

// MCPToolResult is the outcome of a tool invocation.
// Adapters map from mcp.CallToolResult.
type MCPToolResult struct {
	Content           []MCPContentBlock
	StructuredContent json.RawMessage
	IsError           bool
}

// MCPResourceSummary describes an MCP resource listing entry.
type MCPResourceSummary struct {
	URI         string
	Name        string
	Title       string
	Description string
	MIMEType    string
}

// MCPResourceContent is MCP resource payload.
type MCPResourceContent struct {
	URI      string
	MIMEType string
	Text     string
	Binary   []byte
}

// MCPPromptSummary describes an MCP prompt listing entry.
type MCPPromptSummary struct {
	Name        string
	Title       string
	Description string
}

// MCPPromptMessage is one message returned by prompts/get.
type MCPPromptMessage struct {
	Role     string
	Text     string
	MIMEType string
	Binary   []byte
}

// MCPPromptResult is the resolved prompt from prompts/get.
type MCPPromptResult struct {
	Description string
	Messages    []MCPPromptMessage
}

// MCPServer is a connected MCP server an agent can call for tools and context.
// Adapters wrap mcp.ClientSession.
type MCPServer interface {
	Name() string
	Connect(ctx context.Context) error
	Close() error

	ListTools(ctx context.Context) ([]MCPTool, error)
	CallTool(ctx context.Context, name string, args json.RawMessage) (MCPToolResult, error)

	ListResources(ctx context.Context) ([]MCPResourceSummary, error)
	ReadResource(ctx context.Context, uri string) (MCPResourceContent, error)

	ListPrompts(ctx context.Context) ([]MCPPromptSummary, error)
	GetPrompt(ctx context.Context, name string, args map[string]string) (MCPPromptResult, error)
}

// MCPRegistry holds MCP servers available to an agent.
type MCPRegistry interface {
	List() []MCPServerConfig
	Get(name string) (MCPServer, bool)
	ConnectAll(ctx context.Context) error
	CloseAll() error
}

// MCPClientFactory creates MCP server connections from configuration.
// Implementations use github.com/modelcontextprotocol/go-sdk/mcp.Client.
type MCPClientFactory interface {
	NewServer(ctx context.Context, cfg MCPServerConfig) (MCPServer, error)
}
