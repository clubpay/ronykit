package lint

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/ronyup/internal/rkitlint"

	"github.com/spf13/cobra"
)

// Exit codes.
const (
	exitClean    = 0
	exitFindings = 1
	exitError    = 2
	// exitHookFeedback makes Claude Code show stderr to the agent (PostToolUse protocol).
	exitHookFeedback = 2
)

const hookClaudePostEdit = "claude-post-edit"

var opt = struct {
	Fix        bool
	Tests      bool
	AllModules bool
	Hook       string
}{}

func init() {
	flags := Cmd.Flags()
	flags.BoolVar(&opt.Fix, "fix", false, "apply suggested fixes in place")
	flags.BoolVar(&opt.Tests, "tests", false, "also check _test.go files")
	flags.BoolVar(&opt.AllModules, "all-modules", false,
		"also check modules that do not already require x/rkit (skipped by default)")
	flags.StringVar(&opt.Hook, "hook", "",
		"run as an agent hook reading its event from stdin: "+hookClaudePostEdit+
			" (Claude Code PostToolUse; lints the edited file and exits 2 with findings on stderr)")
}

var Cmd = &cobra.Command{
	Use:   "lint [packages]",
	Short: "Report hand-written code that duplicates an x/rkit helper",
	Long: `Report hand-written Go code that duplicates an x/rkit helper:

  must        v, err := f(); if err != nil { panic(err) }   → rkit.Must / rkit.Assert
  mapcollect  loops that only append a map's keys/values      → rkit.MapKeysToArray / rkit.MapToArray
  unsafeconv  hand-rolled zero-copy []byte↔string casts       → rkit.B2S / rkit.S2B
  helper      local copies of rkit.Must, PtrVal, Contains, Coalesce

Packages default to ./... . Modules that do not require x/rkit are skipped unless
--all-modules is set. Silence a deliberate exception with a
//rkitcheck:ignore <reason> comment on its line or the line above.

Exit status: 0 clean, 1 findings left, 2 load or type-check errors.`,
	Run: func(cmd *cobra.Command, args []string) {
		var code int
		if opt.Hook != "" {
			code = runHook(cmd, opt.Hook, cmd.InOrStdin())
		} else {
			code = runLint(cmd, args)
		}

		if code != exitClean {
			os.Exit(code)
		}
	},
}

func runLint(cmd *cobra.Command, patterns []string) int {
	res, err := rkitlint.Run(rkitlint.Options{
		Patterns:   patterns,
		Tests:      opt.Tests,
		AllModules: opt.AllModules,
		Fix:        opt.Fix,
	})
	if err != nil {
		cmd.PrintErrln(err)

		return exitError
	}

	wd, _ := os.Getwd()
	out := cmd.OutOrStdout()

	for _, f := range res.Fixed {
		_, _ = fmt.Fprintln(out, "fixed: "+formatFinding(f, wd))
	}

	fixable := 0

	for _, f := range res.Findings {
		_, _ = fmt.Fprintln(out, formatFinding(f, wd))

		if f.Fixable {
			fixable++
		}
	}

	switch {
	case len(res.Findings) == 0 && len(res.Fixed) == 0:
		return exitClean
	case len(res.Findings) == 0:
		cmd.PrintErrf("ronyup lint: fixed %d finding(s) in %d file(s)\n", len(res.Fixed), len(res.Files))

		return exitClean
	case fixable > 0:
		cmd.PrintErrf("ronyup lint: %d finding(s); %d can be fixed with `ronyup lint --fix`\n",
			len(res.Findings), fixable)
	default:
		cmd.PrintErrf("ronyup lint: %d finding(s)\n", len(res.Findings))
	}

	return exitFindings
}

func runHook(cmd *cobra.Command, hook string, stdin io.Reader) int {
	if hook != hookClaudePostEdit {
		cmd.PrintErrf("unknown --hook %q (supported: %s)\n", hook, hookClaudePostEdit)

		return exitError
	}

	return runClaudePostEdit(cmd, stdin)
}

type postToolUseInput struct {
	ToolInput struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
}

// runClaudePostEdit lints the package of the Go file an agent just edited and reports findings in
// that file only. It fails open: unreadable input, non-Go files and packages that do not load
// (half-finished edits) never produce feedback.
func runClaudePostEdit(cmd *cobra.Command, stdin io.Reader) int {
	var in postToolUseInput
	if err := json.NewDecoder(stdin).Decode(&in); err != nil {
		return exitClean
	}

	file := filepath.Clean(in.ToolInput.FilePath)
	if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
		return exitClean
	}

	res, err := rkitlint.Run(rkitlint.Options{Dir: filepath.Dir(file), Patterns: []string{"."}})
	if err != nil {
		return exitClean
	}

	var lines []string

	for _, f := range res.Findings {
		if sameFile(f.Position.Filename, file) {
			lines = append(lines, formatFinding(f, filepath.Dir(file)))
		}
	}

	if len(lines) == 0 {
		return exitClean
	}

	cmd.PrintErrf("ronyup lint: %s re-implements x/rkit helpers. Use the rkit helper instead "+
		"(`ronyup lint --fix` rewrites the fixable ones):\n%s\n", filepath.Base(file), strings.Join(lines, "\n"))

	return exitHookFeedback
}

func sameFile(a, b string) bool {
	if filepath.Clean(a) == b {
		return true
	}

	ra, errA := filepath.EvalSymlinks(a)
	rb, errB := filepath.EvalSymlinks(b)

	return errA == nil && errB == nil && ra == rb
}

func formatFinding(f rkitlint.Finding, wd string) string {
	name := f.Position.Filename
	if rel, err := filepath.Rel(wd, name); err == nil && !strings.HasPrefix(rel, "..") {
		name = rel
	}

	return fmt.Sprintf(
		"%s:%d:%d: %s (rkitcheck/%s)",
		name,
		f.Position.Line,
		f.Position.Column,
		f.Message,
		f.Category,
	)
}
