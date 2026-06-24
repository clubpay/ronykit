package intent_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
)

type localEcho struct{}

func (localEcho) Definition() intent.ToolDefinition {
	return intent.ToolDefinition{
		Name:        "echo",
		Description: "echo input",
		Parameters: map[string]any{
			"type": "object",
		},
	}
}

func (localEcho) Execute(_ context.Context, args json.RawMessage) (intent.Message, error) {
	return intent.Message{
		Role:  intent.RoleTool,
		Parts: []intent.Part{intent.TextPart(string(args))},
	}, nil
}

type fakeMCPServer struct {
	name string
}

func (f fakeMCPServer) Name() string                  { return f.name }
func (f fakeMCPServer) Connect(context.Context) error { return nil }
func (f fakeMCPServer) Close() error                  { return nil }
func (f fakeMCPServer) ListTools(context.Context) ([]intent.MCPTool, error) {
	return []intent.MCPTool{{
		Name:        "SayHi",
		Description: "say hi",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}}, nil
}
func (f fakeMCPServer) CallTool(_ context.Context, name string, _ json.RawMessage) (intent.MCPToolResult, error) {
	return intent.MCPToolResult{Content: []intent.MCPContentBlock{{Text: "hi from " + name}}}, nil
}
func (f fakeMCPServer) ListResources(context.Context) ([]intent.MCPResourceSummary, error) {
	return nil, nil
}
func (f fakeMCPServer) ReadResource(context.Context, string) (intent.MCPResourceContent, error) {
	return intent.MCPResourceContent{}, nil
}
func (f fakeMCPServer) ListPrompts(context.Context) ([]intent.MCPPromptSummary, error) {
	return nil, nil
}
func (f fakeMCPServer) GetPrompt(context.Context, string, map[string]string) (intent.MCPPromptResult, error) {
	return intent.MCPPromptResult{}, nil
}

func TestRegistryLocalTool(t *testing.T) {
	reg := intent.NewToolRegistry()
	err := reg.Register(localEcho{})
	if err != nil {
		t.Fatal(err)
	}

	defs, err := reg.Definitions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 1 || defs[0].Name != "echo" {
		t.Fatalf("unexpected defs: %#v", defs)
	}

	msg, err := reg.Execute(context.Background(), intent.ToolCall{
		Name:      "echo",
		Arguments: `{"msg":"ok"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Parts[0].Text != `{"msg":"ok"}` {
		t.Fatalf("unexpected tool msg: %#v", msg)
	}
}

func TestRegistryMCPToolQualifiedName(t *testing.T) {
	reg := intent.NewToolRegistry()
	err := reg.RegisterMCP(fakeMCPServer{name: "srv"})
	if err != nil {
		t.Fatal(err)
	}

	msg, err := reg.Execute(context.Background(), intent.ToolCall{
		Name:      "srv/SayHi",
		Arguments: `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Parts[0].Text != "hi from SayHi" {
		t.Fatalf("unexpected msg: %q", msg.Parts[0].Text)
	}
}

func TestRegistryToolNotFound(t *testing.T) {
	reg := intent.NewToolRegistry()
	_, err := reg.Execute(context.Background(), intent.ToolCall{Name: "missing"})
	if !errs.IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestRegistryInvalidLocalName(t *testing.T) {
	reg := intent.NewToolRegistry()
	err := reg.Register(intent.LocalToolFunc{
		Def: intent.ToolDefinition{Name: "bad/name"},
	})
	if err == nil {
		t.Fatal("expected invalid name error")
	}
	if !errors.Is(err, errs.ErrUnsupportedOperation) {
		t.Fatalf("got %v", err)
	}
}
