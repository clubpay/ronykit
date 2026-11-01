package scaffold

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	appscaffold "github.com/clubpay/ronykit/ronyup/internal/scaffold"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestSkillToolsSyncAndValidate(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.work"), []byte("go 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	srv := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "test", Version: "v0"}, nil)
	registerSkillTools(srv)

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

	call := func(name string, args map[string]any) *mcpsdk.CallToolResult {
		t.Helper()

		res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("CallTool(%s): %v", name, err)
		}
		if res.IsError {
			t.Fatalf("CallTool(%s) returned an error: %+v", name, res.Content)
		}

		return res
	}

	args := map[string]any{"workspacePath": root, "skills": []string{"go-modern"}, "overwrite": true}
	syncResult := call("sync_skills", args)
	var syncOut syncSkillsOutput
	decodeStructured(t, syncResult.StructuredContent, &syncOut)
	if !syncOut.Validation.UpToDate {
		t.Fatalf("sync validation should be current: %+v", syncOut.Validation)
	}

	if err := os.WriteFile(
		filepath.Join(root, ".agents", "skills", "go-modern", "SKILL.md"),
		[]byte("stale\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	validateResult := call("validate_skills", args)
	var report appscaffold.SkillsReport
	decodeStructured(t, validateResult.StructuredContent, &report)
	if report.UpToDate {
		t.Fatal("validation should report modified skill content")
	}

	call("sync_skills", args)
	validateResult = call("validate_skills", args)
	decodeStructured(t, validateResult.StructuredContent, &report)
	if !report.UpToDate {
		t.Fatalf("sync should repair reported drift: %+v", report)
	}
}

func decodeStructured(t *testing.T, input, output any) {
	t.Helper()

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, output); err != nil {
		t.Fatalf("decode structured output %T: %v", input, err)
	}
}
