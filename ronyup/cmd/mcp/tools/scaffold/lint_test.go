package scaffold

import (
	"context"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/ronyup/internal/rkitlint/rkitlinttest"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCheckRkitUsage(t *testing.T) {
	dir := rkitlinttest.NewModule(t, true, map[string]string{
		"keys.go": `package lintmod

func keys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}
`,
	})

	ctx := context.Background()
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "v0"}, nil)
	registerCheckRkit(srv)

	st, ct := mcpsdk.NewInMemoryTransports()

	ss, err := srv.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()

	cs, err := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "client", Version: "v0"}, nil).
		Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()

	call := func(args map[string]any) (string, bool) {
		t.Helper()

		res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: "check_rkit_usage", Arguments: args})
		if err != nil {
			t.Fatalf("CallTool: %v", err)
		}

		var text strings.Builder
		for _, c := range res.Content {
			if tc, ok := c.(*mcpsdk.TextContent); ok {
				text.WriteString(tc.Text)
			}
		}

		return text.String(), res.IsError
	}

	if out, isErr := call(map[string]any{"path": dir}); isErr ||
		!strings.Contains(out, "keys.go:4: loop collects map keys") || !strings.Contains(out, "[fixable]") {
		t.Fatalf("check: isError=%v\n%s", isErr, out)
	}

	if out, isErr := call(
		map[string]any{"path": dir, "fix": true},
	); isErr ||
		!strings.Contains(out, "Fixed 1 finding(s)") {
		t.Fatalf("fix: isError=%v\n%s", isErr, out)
	}

	if out, isErr := call(map[string]any{"path": dir}); isErr || !strings.Contains(out, "No findings") {
		t.Fatalf("after fix: isError=%v\n%s", isErr, out)
	}

	if _, isErr := call(map[string]any{"path": ""}); !isErr {
		t.Fatal("empty path must be an error result")
	}
}
