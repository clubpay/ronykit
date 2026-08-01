package mcp

import (
	"context"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestToolInventoryMatchesKnowledgeDocs pins the relationship between the MCP
// tools registered on the server and the knowledge docs under
// knowledge/resources/tools/. Adding a tool without a doc (or a doc without a
// tool) fails this test.
func TestToolInventoryMatchesKnowledgeDocs(t *testing.T) {
	kb := mustLoadKB(t)

	srv := newServer(ServerConfig{
		name:         "ronyup",
		version:      "v0.0.0-test",
		instructions: "test",
		kb:           kb,
	})

	ctx := context.Background()
	st, ct := mcpsdk.NewInMemoryTransports()

	serverSession, err := srv.srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}

	defer serverSession.Close()

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "v0"}, nil)

	clientSession, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}

	defer clientSession.Close()

	res, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	registered := make(map[string]bool, len(res.Tools))
	for _, tool := range res.Tools {
		registered[tool.Name] = true
	}

	if len(registered) == 0 {
		t.Fatal("expected at least one registered tool")
	}

	documented := make(map[string]bool, len(kb.Tools))
	for _, doc := range kb.Tools {
		documented[doc.Name] = true
	}

	for name := range registered {
		if !documented[name] {
			t.Errorf("tool %q is registered but has no knowledge doc at resources/tools/%s.md", name, name)
		}
	}

	for name := range documented {
		if !registered[name] {
			t.Errorf("knowledge doc resources/tools/%s.md exists but no MCP tool %q is registered", name, name)
		}
	}
}
