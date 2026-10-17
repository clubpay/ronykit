package lint

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/ronyup/internal/rkitlint/rkitlinttest"

	"github.com/spf13/cobra"
)

const badSrc = `package hookmod

import "strconv"

func parse() int {
	n, err := strconv.Atoi("1")
	if err != nil {
		panic(err)
	}

	return n + 1
}
`

const goodSrc = `package hookmod

func add(a, b int) int { return a + b }
`

func newHookModule(t *testing.T) string {
	t.Helper()

	return rkitlinttest.NewModule(t, true, map[string]string{
		"bad.go":  badSrc,
		"good.go": goodSrc,
		"README":  "not go",
	})
}

func hookInput(t *testing.T, path string) string {
	t.Helper()

	b, err := json.Marshal(map[string]any{
		"hook_event_name": "PostToolUse",
		"tool_name":       "Edit",
		"tool_input":      map[string]string{"file_path": path},
	})
	if err != nil {
		t.Fatal(err)
	}

	return string(b)
}

func TestClaudePostEditHook(t *testing.T) {
	dir := newHookModule(t)

	tests := []struct {
		name     string
		hook     string
		input    string
		wantCode int
		wantErr  []string
	}{
		{
			"file with findings", hookClaudePostEdit, hookInput(t, filepath.Join(dir, "bad.go")),
			exitHookFeedback,
			[]string{"bad.go", "bad.go:6:2:", "rkit.Must"},
		},
		{"clean file", hookClaudePostEdit, hookInput(t, filepath.Join(dir, "good.go")), exitClean, nil},
		{"non-Go file", hookClaudePostEdit, hookInput(t, filepath.Join(dir, "README")), exitClean, nil},
		{"test file", hookClaudePostEdit, hookInput(t, filepath.Join(dir, "bad_test.go")), exitClean, nil},
		{"unreadable input", hookClaudePostEdit, "{not json", exitClean, nil},
		{"unknown hook", "cursor-whatever", "{}", exitError, []string{"unknown --hook"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer

			cmd := &cobra.Command{}
			cmd.SetErr(&stderr)

			if got := runHook(cmd, tt.hook, strings.NewReader(tt.input)); got != tt.wantCode {
				t.Fatalf("exit code = %d, want %d (stderr: %s)", got, tt.wantCode, stderr.String())
			}

			if len(tt.wantErr) == 0 && stderr.Len() > 0 {
				t.Fatalf("unexpected stderr: %s", stderr.String())
			}

			for _, want := range tt.wantErr {
				if !strings.Contains(stderr.String(), want) {
					t.Errorf("stderr missing %q:\n%s", want, stderr.String())
				}
			}
		})
	}
}
