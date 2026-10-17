package scaffold

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/ronyup/internal/rkitlint"
	"github.com/clubpay/ronykit/x/rkit"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type checkRkitArgs struct {
	Path       string   `json:"path"`
	Patterns   []string `json:"patterns"`
	Fix        bool     `json:"fix"`
	Tests      bool     `json:"tests"`
	AllModules bool     `json:"allModules"`
}

func registerCheckRkit(srv *mcpsdk.Server) {
	tool := &mcpsdk.Tool{
		Name: "check_rkit_usage",
		Description: "Report Go code that re-implements an x/rkit helper (if-err-panic → rkit.Must/Assert, " +
			"map key/value collection loops → rkit.MapKeysToArray/MapToArray, unsafe []byte↔string casts → " +
			"rkit.B2S/S2B, local copies of Must/PtrVal/Contains/Coalesce) and optionally fix it in place. " +
			"Same as `ronyup lint`; `make verify` runs it on every feature module.",
		InputSchema: map[string]any{
			schemaKeyType: schemaKeyObject,
			schemaKeyProperties: map[string]any{
				argPath: stringArg("Directory to resolve package patterns from: a feature module or " +
					"the Go workspace root."),
				"patterns": stringSliceArg("Go package patterns relative to path. Defaults to [\"./...\"]."),
				"fix":      boolArg("Rewrite fixable findings in place (adds the rkit import)."),
				"tests":    boolArg("Also check _test.go files."),
				"allModules": boolArg("Also check modules that do not already require x/rkit " +
					"(skipped by default)."),
			},
			schemaKeyRequired: []string{argPath},
		},
	}

	mcpsdk.AddTool(srv, tool,
		func(_ context.Context, _ *mcpsdk.CallToolRequest, args checkRkitArgs) (
			*mcpsdk.CallToolResult, any, error,
		) {
			if args.Path == "" {
				return errorResult(rkit.L("path is required")), nil, nil
			}

			dir, err := filepath.Abs(args.Path)
			if err != nil {
				return errorResult(rkit.L("invalid path %s: %v", args.Path, err)), nil, nil
			}

			res, err := rkitlint.Run(rkitlint.Options{
				Dir:        dir,
				Patterns:   args.Patterns,
				Tests:      args.Tests,
				AllModules: args.AllModules,
				Fix:        args.Fix,
			})

			var loadErr *rkitlint.LoadError

			switch {
			case errors.As(err, &loadErr):
				return errorResult(
					rkit.L("Packages under %s do not compile; fix these first:", dir),
					rkit.L("%s", strings.Join(loadErr.Errors, "\n")),
				), nil, nil
			case err != nil:
				return errorResult(rkit.L("check_rkit_usage failed: %v", err)), nil, nil
			}

			return textResult(checkRkitLines(dir, res)...), nil, nil
		})
}

func checkRkitLines(dir string, res *rkitlint.Result) []rkit.StrLine {
	if len(res.Findings) == 0 && len(res.Fixed) == 0 {
		return []rkit.StrLine{rkit.L("No findings: no hand-written code duplicates an x/rkit helper.")}
	}

	var lines []rkit.StrLine

	if len(res.Fixed) > 0 {
		lines = append(lines, rkit.L("Fixed %d finding(s) in %d file(s):", len(res.Fixed), len(res.Files)))
		for _, f := range res.Fixed {
			lines = append(lines, rkit.L("- %s", relFinding(dir, f)))
		}
	}

	if len(res.Findings) > 0 {
		lines = append(lines, rkit.L("%d finding(s) to fix by hand or with fix=true:", len(res.Findings)))
		for _, f := range res.Findings {
			fixable := ""
			if f.Fixable {
				fixable = " [fixable]"
			}

			lines = append(lines, rkit.L("- %s%s", relFinding(dir, f), fixable))
		}
	}

	return lines
}

func relFinding(dir string, f rkitlint.Finding) string {
	name := f.Position.Filename
	if rel, err := filepath.Rel(dir, name); err == nil && !strings.HasPrefix(rel, "..") {
		name = rel
	}

	return name + ":" + rkit.IntToStr(f.Position.Line) + ": " + f.Message + " (rkitcheck/" + f.Category + ")"
}
