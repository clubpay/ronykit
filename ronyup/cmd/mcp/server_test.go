package mcp

import (
	"testing"

	"github.com/clubpay/ronykit/ronyup/cmd/mcp/knowledge"
)

func TestNewServer_DoesNotPanicOnSchemaTags(t *testing.T) {
	kb := mustLoadKB(t)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("newServer panicked: %v", r)
		}
	}()

	_ = newServer(ServerConfig{
		name:         "ronyup",
		version:      "v0.0.0-test",
		instructions: "test",
		kb:           kb,
	})
}

func mustLoadKB(t *testing.T) *knowledge.Base {
	t.Helper()

	kb, err := knowledge.Load()
	if err != nil {
		t.Fatalf("knowledge.Load() failed: %v", err)
	}

	return kb
}
