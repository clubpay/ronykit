package rkitlint

import (
	"bytes"
	"cmp"
	"fmt"
	"go/format"
	"go/token"
	"os"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/analysis"
	gochecker "golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

// Options configure Run.
type Options struct {
	// Dir is the directory package patterns are resolved from; "" means the current directory.
	Dir string
	// Patterns are go package patterns; empty means "./...".
	Patterns []string
	// Tests also checks _test.go files.
	Tests bool
	// AllModules also checks modules that do not already require x/rkit. By default those are
	// skipped: rkit is not worth a new dependency for a one-line helper.
	AllModules bool
	// Fix applies suggested fixes in place.
	Fix bool
}

// Finding is one report.
type Finding struct {
	Position token.Position
	Category string
	Message  string
	Fixable  bool
}

func (f Finding) String() string {
	return fmt.Sprintf("%s: %s (rkitcheck/%s)", f.Position, f.Message, f.Category)
}

// Result is the outcome of Run.
type Result struct {
	// Findings are the reports still present in the code: all of them, or with Fix set, the ones
	// without a fix (or whose fix conflicted with another).
	Findings []Finding
	// Fixed are the reports whose fixes were applied (Fix only).
	Fixed []Finding
	// Files are the files rewritten (Fix only).
	Files []string
}

// LoadError means the target packages could not be loaded or type-checked.
type LoadError struct {
	Errors []string
}

func (e *LoadError) Error() string {
	return "packages contain errors:\n" + strings.Join(e.Errors, "\n")
}

// Run loads the packages matching opts.Patterns, runs Analyzer on those whose module requires
// x/rkit (unless opts.AllModules), and applies the suggested fixes when opts.Fix is set.
func Run(opts Options) (*Result, error) {
	patterns := opts.Patterns
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode:  packages.LoadSyntax | packages.NeedModule,
		Dir:   opts.Dir,
		Tests: opts.Tests,
	}, patterns...)
	if err != nil {
		return nil, err
	}

	var (
		targets []*packages.Package
		errs    []string
		mods    = make(map[string]bool)
	)

	for _, p := range pkgs {
		if !opts.AllModules && !requiresRkit(p.Module, mods) {
			continue
		}

		if len(p.Errors) > 0 {
			for _, e := range p.Errors {
				errs = append(errs, e.Error())
			}

			continue
		}

		targets = append(targets, p)
	}

	if len(errs) > 0 {
		return nil, &LoadError{Errors: errs}
	}

	if len(targets) == 0 {
		return &Result{}, nil
	}

	graph, err := gochecker.Analyze([]*analysis.Analyzer{Analyzer}, targets, nil)
	if err != nil {
		return nil, err
	}

	diags, err := collect(graph)
	if err != nil {
		return nil, err
	}

	res := &Result{}

	if !opts.Fix {
		for _, d := range diags {
			res.Findings = append(res.Findings, d.finding)
		}

		return res, nil
	}

	return res, applyFixes(diags, res)
}

type diag struct {
	finding Finding
	edits   []fileEdit
}

type fileEdit struct {
	file       string
	start, end int
	text       []byte
}

func collect(graph *gochecker.Graph) ([]diag, error) {
	type key struct {
		file string
		off  int
		msg  string
	}

	var (
		out  []diag
		seen = make(map[key]bool)
	)

	for _, act := range graph.Roots {
		if act.Err != nil {
			return nil, fmt.Errorf("%s: %w", act.Package.PkgPath, act.Err)
		}

		fset := act.Package.Fset

		for _, d := range act.Diagnostics {
			pos := fset.Position(d.Pos)

			// With Tests set, a file appears in both the package and its test variant.
			k := key{pos.Filename, pos.Offset, d.Message}
			if seen[k] {
				continue
			}

			seen[k] = true

			dg := diag{finding: Finding{Position: pos, Category: d.Category, Message: d.Message}}
			if len(d.SuggestedFixes) > 0 {
				dg.finding.Fixable = true

				for _, e := range d.SuggestedFixes[0].TextEdits {
					tf := fset.File(e.Pos)

					end := e.End
					if !end.IsValid() {
						end = e.Pos
					}

					dg.edits = append(dg.edits, fileEdit{
						file:  tf.Name(),
						start: tf.Offset(e.Pos),
						end:   tf.Offset(end),
						text:  e.NewText,
					})
				}
			}

			out = append(out, dg)
		}
	}

	slices.SortFunc(out, func(a, b diag) int {
		return cmp.Or(
			cmp.Compare(a.finding.Position.Filename, b.finding.Position.Filename),
			cmp.Compare(a.finding.Position.Offset, b.finding.Position.Offset),
		)
	})

	return out, nil
}

// applyFixes applies each diagnostic's edits unless they conflict with an edit already accepted;
// identical edits (the shared rkit import) are merged.
func applyFixes(diags []diag, res *Result) error {
	accepted := make(map[string][]fileEdit)

	for _, d := range diags {
		if len(d.edits) == 0 || conflicts(accepted, d.edits) {
			res.Findings = append(res.Findings, d.finding)

			continue
		}

		for _, e := range d.edits {
			if !slices.ContainsFunc(accepted[e.file], e.same) {
				accepted[e.file] = append(accepted[e.file], e)
			}
		}

		res.Fixed = append(res.Fixed, d.finding)
	}

	for file, edits := range accepted {
		if err := rewrite(file, edits); err != nil {
			return err
		}

		res.Files = append(res.Files, file)
	}

	slices.Sort(res.Files)

	return nil
}

func (e fileEdit) same(o fileEdit) bool {
	return e.file == o.file && e.start == o.start && e.end == o.end && bytes.Equal(e.text, o.text)
}

func conflicts(accepted map[string][]fileEdit, edits []fileEdit) bool {
	for _, e := range edits {
		for _, o := range accepted[e.file] {
			if e.same(o) {
				continue
			}

			overlap := e.start < o.end && o.start < e.end
			sameInsert := e.start == e.end && o.start == o.end && e.start == o.start

			if overlap || sameInsert {
				return true
			}
		}
	}

	return false
}

func rewrite(file string, edits []fileEdit) error {
	src, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	slices.SortFunc(edits, func(a, b fileEdit) int {
		return cmp.Or(cmp.Compare(a.start, b.start), cmp.Compare(a.end, b.end))
	})

	var (
		buf  bytes.Buffer
		last int
	)

	for _, e := range edits {
		buf.Write(src[last:e.start])
		buf.Write(e.text)
		last = e.end
	}

	buf.Write(src[last:])

	out, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("%s: fixed source does not parse: %w", file, err)
	}

	info, err := os.Stat(file)
	if err != nil {
		return err
	}

	return os.WriteFile(file, out, info.Mode().Perm())
}

// requiresRkit reports whether mod's go.mod requires x/rkit. Results are cached by go.mod path.
func requiresRkit(mod *packages.Module, cache map[string]bool) bool {
	if mod == nil || mod.GoMod == "" || mod.Path == RkitPath {
		return false
	}

	if v, ok := cache[mod.GoMod]; ok {
		return v
	}

	cache[mod.GoMod] = false

	data, err := os.ReadFile(mod.GoMod)
	if err != nil {
		return false
	}

	mf, err := modfile.ParseLax(mod.GoMod, data, nil)
	if err != nil {
		return false
	}

	for _, r := range mf.Require {
		if r.Mod.Path == RkitPath {
			cache[mod.GoMod] = true

			return true
		}
	}

	return false
}
