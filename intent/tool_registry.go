package intent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/clubpay/ronykit/intent/errs"
)

const qualifiedSep = "/"

// DefaultToolRegistry stores local tools and MCP server tools.
type DefaultToolRegistry struct {
	mu    sync.RWMutex
	local map[string]LocalTool
	mcp   map[string]mcpToolBinding
}

type mcpToolBinding struct {
	server MCPServer
	tool   MCPTool
}

// NewToolRegistry returns an empty tool registry.
func NewToolRegistry() *DefaultToolRegistry {
	return &DefaultToolRegistry{
		local: make(map[string]LocalTool),
		mcp:   make(map[string]mcpToolBinding),
	}
}

func (r *DefaultToolRegistry) Register(t LocalTool) error {
	if r == nil {
		return errs.Wrap(errs.ErrUnsupportedOperation, "tool registry is nil")
	}

	if t == nil {
		return errs.Wrap(errs.ErrUnsupportedOperation, "tool is nil")
	}

	def := t.Definition()
	if def.Name == "" {
		return errs.Wrap(errs.ErrUnsupportedOperation, "tool name is required")
	}

	if strings.Contains(def.Name, qualifiedSep) {
		return errs.Wrap(errs.ErrUnsupportedOperation, "local tool name must not contain '/'")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.local[def.Name] = t

	return nil
}

func (r *DefaultToolRegistry) RegisterMCP(server MCPServer) error {
	if r == nil {
		return errs.Wrap(errs.ErrUnsupportedOperation, "tool registry is nil")
	}

	if server == nil {
		return errs.Wrap(errs.ErrUnsupportedOperation, "mcp server is nil")
	}

	if server.Name() == "" {
		return errs.Wrap(errs.ErrUnsupportedOperation, "mcp server name is required")
	}

	tools, err := server.ListTools(context.Background())
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, t := range tools {
		key := qualifyTool(server.Name(), t.Name)
		r.mcp[key] = mcpToolBinding{server: server, tool: t}
	}

	return nil
}

func (r *DefaultToolRegistry) Definitions(ctx context.Context) ([]ToolDefinition, error) {
	if r == nil {
		return nil, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]ToolDefinition, 0, len(r.local)+len(r.mcp))
	for _, t := range r.local {
		out = append(out, t.Definition())
	}

	for key, binding := range r.mcp {
		params := any(map[string]any{})
		if len(binding.tool.InputSchema) > 0 {
			err := json.Unmarshal(binding.tool.InputSchema, &params)
			if err != nil {
				return nil, fmt.Errorf("decode mcp tool schema for %q: %w", key, err)
			}
		}

		out = append(out, ToolDefinition{
			Name:        key,
			Description: binding.tool.Description,
			Parameters:  params,
		})
	}

	_ = ctx

	return out, nil
}

func (r *DefaultToolRegistry) Execute(ctx context.Context, call ToolCall) (Message, error) {
	if r == nil {
		return Message{}, errs.Wrap(errs.ErrUnsupportedOperation, "tool registry is nil")
	}

	if call.Name == "" {
		return Message{}, errs.Wrap(errs.ErrToolNotFound, "tool call name is empty")
	}

	r.mu.RLock()
	localTool, localOK := r.local[call.Name]
	mcpBinding, mcpOK := r.mcp[call.Name]
	r.mu.RUnlock()

	switch {
	case localOK:
		return localTool.Execute(ctx, json.RawMessage(call.Arguments))
	case mcpOK:
		return r.executeMCP(ctx, call, mcpBinding)
	default:
		return Message{}, errs.ToolNotFound(call.Name)
	}
}

func (r *DefaultToolRegistry) executeMCP(
	ctx context.Context,
	call ToolCall,
	binding mcpToolBinding,
) (Message, error) {
	result, err := binding.server.CallTool(ctx, binding.tool.Name, json.RawMessage(call.Arguments))
	if err != nil {
		return Message{}, err
	}

	text := formatMCPToolResult(result)

	msg := Message{
		Role:       RoleTool,
		Parts:      []Part{TextPart(text)},
		ToolCallID: call.ID,
		ToolName:   call.Name,
	}
	if result.IsError {
		return msg, fmt.Errorf("tool %q returned error: %s", call.Name, text)
	}

	return msg, nil
}

func formatMCPToolResult(result MCPToolResult) string {
	if len(result.StructuredContent) > 0 {
		return string(result.StructuredContent)
	}

	var b strings.Builder

	for _, block := range result.Content {
		if block.Text != "" {
			b.WriteString(block.Text)
		}
	}

	return b.String()
}

func qualifyTool(serverName, toolName string) string {
	return serverName + qualifiedSep + toolName
}

var _ ToolRegistry = (*DefaultToolRegistry)(nil)
