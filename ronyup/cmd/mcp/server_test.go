package mcp

import (
	"testing"

	"github.com/clubpay/ronykit/ronyup/cmd/mcp/knowledge"
	"github.com/clubpay/ronykit/ronyup/internal"
)

func TestNormalizeRelativePath(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		expectErr bool
	}{
		{name: "simple", input: "billing", want: "billing"},
		{name: "nested", input: "billing/invoice", want: "billing/invoice"},
		{name: "absolute gets normalized", input: "/billing", want: "billing"},
		{name: "traversal rejected", input: "../billing", expectErr: true},
		{name: "empty rejected", input: "   ", expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeRelativePath(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestNormalizeFeatureName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		expectErr bool
	}{
		{name: "valid", input: "billing", want: "billing"},
		{name: "underscore valid", input: "billing_v2", want: "billing_v2"},
		{name: "invalid space", input: "billing service", expectErr: true},
		{name: "invalid hyphen", input: "billing-service", expectErr: true},
		{name: "empty", input: "", expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeFeatureName(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got=%q want=%q", got, tc.want)
			}
		})
	}
}

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
		skeletonFS:   internal.Skeleton,
		cmdRunner:    defaultRunner{},
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
