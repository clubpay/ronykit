package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/x/rkit"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// designDir is the workspace-relative directory that holds SRS/SDD documents.
const designDir = "docs/design"

// designDocSpec describes one required design document.
type designDocSpec struct {
	kind   string // human-readable label, e.g. "SRS"
	suffix string // file suffix, e.g. "-srs.md"
}

var requiredDesignDocs = []designDocSpec{
	{kind: "SRS", suffix: "-srs.md"},
	{kind: "SDD", suffix: "-sdd.md"},
}

// checkDesignGate verifies that approved SRS and SDD documents exist for the
// feature before scaffolding (and therefore implementation) is allowed. It
// returns a list of human-readable problems; an empty slice means the gate
// passes.
//
// A document satisfies the gate only when it exists at
// docs/design/<feature><suffix> and its YAML frontmatter declares
// `status: approved`. The "approved" status is expected to be set by the user
// after review — agents write documents with `status: draft`.
func checkDesignGate(workspacePath, feature string) []string {
	var problems []string

	baseDir := designBaseDir(workspacePath)

	for _, spec := range requiredDesignDocs {
		rel := filepath.ToSlash(filepath.Join(designDir, feature+spec.suffix))
		full := filepath.Join(baseDir, designDir, feature+spec.suffix)

		data, err := os.ReadFile(full)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s not found at %s — write it first.", spec.kind, rel))

			continue
		}

		status := frontmatterStatus(string(data))
		switch {
		case status == "":
			problems = append(problems, fmt.Sprintf(
				"%s at %s has no `status:` frontmatter — expected `status: approved` after user review.",
				spec.kind, rel,
			))
		case !strings.EqualFold(status, "approved"):
			problems = append(problems, fmt.Sprintf(
				"%s at %s is not approved (status: %q) — the user must approve it (set `status: approved`).",
				spec.kind, rel, status,
			))
		}
	}

	return problems
}

// designBaseDir returns the directory that contains the docs/design tree.
//
// In a fullstack scaffold the Go workspace (go.work) lives under <root>/backend
// while docs/ stays at <root>. scaffold_feature is invoked from the workspace
// (backend) directory, so when docs/design is not present there we fall back to
// the parent directory.
func designBaseDir(workspacePath string) string {
	if _, err := os.Stat(filepath.Join(workspacePath, designDir)); err == nil {
		return workspacePath
	}

	parent := filepath.Dir(workspacePath)
	if parent != workspacePath {
		if _, err := os.Stat(filepath.Join(parent, designDir)); err == nil {
			return parent
		}
	}

	return workspacePath
}

// designGateError builds an actionable error result explaining why
// scaffolding is blocked and how to unblock it.
func designGateError(feature string, problems []string) *mcpsdk.CallToolResult {
	const approveHint = "set frontmatter `status: approved` after user review"

	lines := make([]rkit.StrLine, 0, len(problems)+9)
	lines = append(lines,
		rkit.L(
			"Design gate failed for feature %q. Scaffolding and implementation are blocked until an approved SRS and SDD exist.",
			feature,
		),
		rkit.L(""),
		rkit.L("Problems:"),
	)

	for _, p := range problems {
		lines = append(lines, rkit.L("- %s", p))
	}

	lines = append(lines,
		rkit.L(""),
		rkit.L("How to proceed:"),
		rkit.L("1. Run the `write-srs` prompt; the user approves it (%s).", approveHint),
		rkit.L("2. Run the `write-sdd` prompt; the user approves it (%s).", approveHint),
		rkit.L("3. Re-run scaffold_feature."),
		rkit.L(""),
		rkit.L("Only set skipDesignGate=true if the user explicitly asked to skip the design documents."),
	)

	return errorResult(lines...)
}

// frontmatterStatus extracts the `status` value from a leading YAML
// frontmatter block. It returns an empty string when there is no frontmatter
// or no `status` key.
func frontmatterStatus(content string) string {
	content = strings.TrimPrefix(content, "\ufeff")

	lines := strings.Split(content, "\n")

	i := 0
	for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
		i++
	}

	if i >= len(lines) || strings.TrimSpace(lines[i]) != "---" {
		return ""
	}

	i++

	for ; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "---" || line == "..." {
			break
		}

		key, val, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != "status" {
			continue
		}

		v := strings.TrimSpace(val)
		if idx := strings.IndexByte(v, '#'); idx >= 0 {
			v = strings.TrimSpace(v[:idx])
		}

		return strings.Trim(v, `"'`)
	}

	return ""
}
