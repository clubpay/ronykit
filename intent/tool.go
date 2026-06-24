package intent

import (
	"context"
	"encoding/json"
)

// LocalTool is a local executable tool registered on the agent.
type LocalTool interface {
	Definition() ToolDefinition
	Execute(ctx context.Context, args json.RawMessage) (Message, error)
}

// LocalToolFunc adapts functions to LocalTool.
type LocalToolFunc struct {
	Def ToolDefinition
	Fn  func(ctx context.Context, args json.RawMessage) (Message, error)
}

func (t LocalToolFunc) Definition() ToolDefinition { return t.Def }

func (t LocalToolFunc) Execute(ctx context.Context, args json.RawMessage) (Message, error) {
	if t.Fn == nil {
		return Message{}, nil
	}

	return t.Fn(ctx, args)
}

// ToolExecutor resolves and runs tool calls.
type ToolExecutor interface {
	Definitions(ctx context.Context) ([]ToolDefinition, error)
	Execute(ctx context.Context, call ToolCall) (Message, error)
}

// ToolRegistry holds local and MCP-backed tools.
type ToolRegistry interface {
	ToolExecutor
	Register(t LocalTool) error
	RegisterMCP(server MCPServer) error
}
