package scaffold

import (
	"context"
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

type checkFrontendDesignArgs struct {
	WorkspacePath string `json:"workspacePath"`
	App           string `json:"app"`
}

var requiredFrontendSections = []string{
	"Context",
	"Aesthetic direction",
	"Token plan",
	"Layout",
	"Component rules",
	"Motion",
	"v1 surfaces",
	"Quality floor",
}

func registerCheckFrontendDesign(srv *mcpsdk.Server) {
	tool := &mcpsdk.Tool{
		Name: "check_frontend_design",
		Description: "Verify that the frontend design document (docs/design/<app>-frontend-design.md) " +
			"exists, has frontmatter `status: approved`, has no leftover template comments, and includes " +
			"all required design-system sections before initializing a frontend stack or writing UI code.",
		InputSchema: map[string]any{
			schemaKeyType: schemaKeyObject,
			schemaKeyProperties: map[string]any{
				argWorkspacePath: stringArg(
					"The path to the workspace root (or frontend/ / backend/ subdirectory).",
				),
				"app": map[string]any{
					schemaKeyType: schemaTypeString,
					schemaKeyDescription: "Frontend app slug ('web' for a single app at frontend/, " +
						"or the directory name for frontend/<app>/).",
					schemaKeyDefault: "web",
				},
			},
			schemaKeyRequired: []string{argWorkspacePath},
		},
	}

	mcpsdk.AddTool(srv, tool,
		func(_ context.Context, _ *mcpsdk.CallToolRequest, args checkFrontendDesignArgs) (
			*mcpsdk.CallToolResult, any, error,
		) {
			if args.WorkspacePath == "" {
				return errorResult(rkit.L("workspacePath is required")), nil, nil
			}

			app := rkit.Coalesce(strings.TrimSpace(args.App), "web")

			problems := checkFrontendDesignGate(args.WorkspacePath, app)
			if len(problems) > 0 {
				return frontendDesignGateError(app, problems), nil, nil
			}

			rel := filepath.ToSlash(filepath.Join(designDir, app+"-frontend-design.md"))

			return textResult(
				rkit.L("Frontend design gate passed for app %q (%s is approved and complete).", app, rel),
			), nil, nil
		})
}

func checkFrontendDesignGate(workspacePath, app string) []string {
	app = rkit.Coalesce(strings.TrimSpace(app), "web")
	baseDir := designBaseDir(workspacePath)
	rel := filepath.ToSlash(filepath.Join(designDir, app+"-frontend-design.md"))
	full := filepath.Join(baseDir, designDir, app+"-frontend-design.md")

	data, err := os.ReadFile(full)
	if err != nil {
		return []string{
			fmt.Sprintf(
				"Frontend design doc not found at %s — write it first using the `design-frontend` prompt.",
				rel,
			),
		}
	}

	content := string(data)

	var problems []string

	status := frontmatterStatus(content)
	switch {
	case status == "":
		problems = append(problems, fmt.Sprintf(
			"Frontend design doc at %s has no `status:` frontmatter — expected `status: approved` after user review.",
			rel,
		))
	case !strings.EqualFold(status, "approved"):
		problems = append(problems, fmt.Sprintf(
			"Frontend design doc at %s is not approved (status: %q) — the user must approve it (set `status: approved`).",
			rel,
			status,
		))
	}

	lowerContent := strings.ToLower(content)
	for _, section := range requiredFrontendSections {
		if !hasMarkdownSection(lowerContent, strings.ToLower(section)) {
			problems = append(problems, fmt.Sprintf(
				"Frontend design doc at %s is missing required section %q "+
					"(see `knowledge://ronyup/architecture/frontend-design-template`).",
				rel,
				section,
			))
		}
	}

	if strings.Contains(content, "<!--") {
		problems = append(problems, fmt.Sprintf(
			"Frontend design doc at %s still contains unfilled `<!-- ... -->` "+
				"instructional comments — replace all placeholders before approval.",
			rel,
		))
	}

	return problems
}

func hasMarkdownSection(lowerContent, lowerTitle string) bool {
	for line := range strings.SplitSeq(lowerContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") && strings.Contains(trimmed, lowerTitle) {
			return true
		}
	}

	return false
}

func frontendDesignGateError(app string, problems []string) *mcpsdk.CallToolResult {
	lines := make([]rkit.StrLine, 0, len(problems)+9)
	lines = append(lines,
		rkit.L(
			"Frontend design gate failed for app %q. Frontend initialization and UI "+
				"implementation are blocked until the design doc is complete and approved.",
			app,
		),
		rkit.L(""),
		rkit.L("Problems:"),
	)

	for _, p := range problems {
		lines = append(lines, rkit.L("- %s", p))
	}

	lines = append(
		lines,
		rkit.L(""),
		rkit.L("How to proceed:"),
		rkit.L(
			"1. Read skills `frontend-design`, `design-tokens`, `typography`, `ui-polish` "+
				"and `knowledge://ronyup/architecture/frontend-design-template`.",
		),
		rkit.L(
			"2. Write or complete `docs/design/%s-frontend-design.md` with `status: draft` "+
				"(remove all `<!-- ... -->` comments).",
			app,
		),
		rkit.L("3. Present the summary to the user so they can review and set `status: approved`."),
		rkit.L(
			"4. Re-run `check_frontend_design` before running `create-next-app`, "+
				"`create vite`, or writing UI components.",
		),
	)

	return errorResult(lines...)
}
