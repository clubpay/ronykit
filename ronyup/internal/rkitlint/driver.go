package rkitlint

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/clubpay/ronykit/x/rkit"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/analysis"
	gochecker "golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"
)

// maxFixRounds bounds re-analysis when fixes overlap (e.g. a call rewritten inside a statement
// that another rule replaces): each round applies the fixes that do not conflict.
const maxFixRounds = 3

// Options configure Run.
type Options struct {
	// Dir is the directory package patterns are resolved from; "" means the current directory.
	Dir string
	// Patterns are go package patterns; empty means "./...".
	Patterns []string
	// Tests also checks _test.go files.
	Tests bool
	// AllModules also checks modules that do not already require x/rkit. By default those are
	// skipped: a module opts in to rkit by requiring it.
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
	// without a fix (or whose fix kept conflicting with another).
	Findings []Finding
	// Fixed are the reports whose fixes were applied (Fix only).
	Fixed []Finding
	// Files are the files rewritten (Fix only).
	Files []string
	// NeedsTidy lists module directories whose go.mod does not require x/rkit but whose code now
	// imports it (Fix with AllModules only); run `go mod tidy` there.
	NeedsTidy []string
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
	res := &Result{}
	files := make(map[string]bool)
	tidy := make(map[string]bool)

	for round := 1; ; round++ {
		an, err := analyze(opts)
		if err != nil {
			// After the first round, a module that now imports rkit without requiring it may not
			// load until `go mod tidy` runs (NeedsTidy); keep what was fixed.
			var loadErr *LoadError
			if round > 1 && errors.As(err, &loadErr) {
				break
			}

			return nil, err
		}

		if !opts.Fix {
			for _, d := range an.diags {
				res.Findings = append(res.Findings, d.finding)
			}

			return res, nil
		}

		fixed, left, err := applyFixes(an, files, tidy)
		if err != nil {
			return nil, err
		}

		res.Fixed = append(res.Fixed, fixed...)
		res.Findings = left

		if len(fixed) == 0 || round == maxFixRounds || !slices.ContainsFunc(left, isFixable) {
			break
		}
	}

	for f := range files {
		res.Files = append(res.Files, f)
	}

	for m := range tidy {
		res.NeedsTidy = append(res.NeedsTidy, m)
	}

	slices.Sort(res.Files)
	slices.Sort(res.NeedsTidy)

	return res, nil
}

func isFixable(f Finding) bool { return f.Fixable }

type analysisRun struct {
	diags []diag
	// importNames maps file → import path → the name the file refers to it by.
	importNames map[string]map[string]string
}

type diag struct {
	finding Finding
	edits   []fileEdit
	// moduleDir is set when the module does not require x/rkit yet.
	untidyModule string
}

type fileEdit struct {
	file       string
	start, end int
	text       []byte
}

func analyze(opts Options) (*analysisRun, error) {
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
		if p.Module != nil && p.Module.Path == RkitPath {
			continue
		}

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

	an := &analysisRun{importNames: make(map[string]map[string]string)}
	if len(targets) == 0 {
		return an, nil
	}

	graph, err := gochecker.Analyze([]*analysis.Analyzer{Analyzer}, targets, nil)
	if err != nil {
		return nil, err
	}

	if err := an.collect(graph, mods); err != nil {
		return nil, err
	}

	return an, nil
}

func (an *analysisRun) collect(graph *gochecker.Graph, mods map[string]bool) error {
	type key struct {
		file string
		off  int
		msg  string
	}

	seen := make(map[key]bool)

	for _, act := range graph.Roots {
		if act.Err != nil {
			return fmt.Errorf("%s: %w", act.Package.PkgPath, act.Err)
		}

		pkg := act.Package
		fset := pkg.Fset

		untidy := ""
		if pkg.Module != nil && !requiresRkit(pkg.Module, mods) {
			untidy = pkg.Module.Dir
		}

		for _, d := range act.Diagnostics {
			pos := fset.Position(d.Pos)

			// With Tests set, a file appears in both the package and its test variant.
			k := key{pos.Filename, pos.Offset, d.Message}
			if seen[k] {
				continue
			}

			seen[k] = true

			dg := diag{
				finding:      Finding{Position: pos, Category: d.Category, Message: d.Message},
				untidyModule: untidy,
			}

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

			an.diags = append(an.diags, dg)
		}

		for _, f := range pkg.Syntax {
			name := fset.File(f.FileStart).Name()
			if an.importNames[name] != nil {
				continue
			}

			names := make(map[string]string)

			for _, imp := range f.Imports {
				if pn := pkg.TypesInfo.PkgNameOf(imp); pn != nil {
					names[pn.Imported().Path()] = pn.Name()
				}
			}

			an.importNames[name] = names
		}
	}

	slices.SortFunc(an.diags, func(a, b diag) int {
		return rkit.Coalesce(
			cmp.Compare(a.finding.Position.Filename, b.finding.Position.Filename),
			cmp.Compare(a.finding.Position.Offset, b.finding.Position.Offset),
		)
	})

	return nil
}

// applyFixes applies each diagnostic's edits unless they conflict with an edit already accepted;
// identical edits (the shared rkit import) are merged. Rewritten files are added to files, and
// modules that now import rkit without requiring it to tidy.
func applyFixes(an *analysisRun, files, tidy map[string]bool) (fixed, left []Finding, err error) {
	accepted := make(map[string][]fileEdit)

	for _, d := range an.diags {
		if len(d.edits) == 0 || conflicts(accepted, d.edits) {
			left = append(left, d.finding)

			continue
		}

		for _, e := range d.edits {
			if !slices.ContainsFunc(accepted[e.file], e.same) {
				accepted[e.file] = append(accepted[e.file], e)
			}
		}

		fixed = append(fixed, d.finding)

		if d.untidyModule != "" {
			tidy[d.untidyModule] = true
		}
	}

	for file, edits := range accepted {
		if err := rewrite(file, edits, an.importNames[file]); err != nil {
			return nil, nil, err
		}

		files[file] = true
	}

	return fixed, left, nil
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

func rewrite(file string, edits []fileEdit, importNames map[string]string) error {
	src, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	slices.SortFunc(edits, func(a, b fileEdit) int {
		return rkit.Coalesce(cmp.Compare(a.start, b.start), cmp.Compare(a.end, b.end))
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

	out, err := dropUnusedImports(file, buf.Bytes(), importNames)
	if err != nil {
		return err
	}

	info, err := os.Stat(file)
	if err != nil {
		return err
	}

	return os.WriteFile(file, out, info.Mode().Perm())
}

// dropUnusedImports removes the imports that fixes left unused (say, the last slices.Contains
// became rkit.Contains) and formats the result. importNames gives the names imports were referred
// to by before the fix.
func dropUnusedImports(file string, src []byte, importNames map[string]string) ([]byte, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("%s: fixed source does not parse: %w", filepath.Base(file), err)
	}

	used := make(map[string]bool)

	ast.Inspect(f, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			// Obj is nil for identifiers the parser could not resolve in the file: package names.
			if id, isIdent := sel.X.(*ast.Ident); isIdent && id.Obj == nil { //nolint:staticcheck
				used[id.Name] = true
			}
		}

		return true
	})

	unused := func(imp *ast.ImportSpec) bool {
		path := rkit.Ok(strconv.Unquote(imp.Path.Value))

		name := importNames[path]
		if imp.Name != nil {
			name = imp.Name.Name
		}

		if name == "" && path == RkitPath {
			name = "rkit"
		}

		return name != "" && name != "_" && name != "." && path != "C" && !used[name]
	}

	tf := fset.File(f.FileStart)

	var cuts [][2]int

	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}

		for _, spec := range gen.Specs {
			imp, isImport := spec.(*ast.ImportSpec)
			if !isImport || !unused(imp) {
				continue
			}

			var node ast.Node = imp
			if !gen.Lparen.IsValid() {
				node = gen // `import "x"`: drop the whole declaration
			}

			cuts = append(cuts, lineSpan(src, tf.Offset(node.Pos()), tf.Offset(node.End())))
		}
	}

	for _, cut := range slices.Backward(cuts) {
		src = append(src[:cut[0]:cut[0]], src[cut[1]:]...)
	}

	out, err := format.Source(src)
	if err != nil {
		return nil, fmt.Errorf("%s: fixed source does not format: %w", filepath.Base(file), err)
	}

	return out, nil
}

// lineSpan widens [start, end) to whole lines (including a trailing comment and the newline) when
// nothing else shares them.
func lineSpan(src []byte, start, end int) [2]int {
	s := start
	for s > 0 && (src[s-1] == ' ' || src[s-1] == '\t') {
		s--
	}

	if s > 0 && src[s-1] != '\n' {
		return [2]int{start, end}
	}

	if nl := bytes.IndexByte(src[end:], '\n'); nl >= 0 {
		return [2]int{s, end + nl + 1}
	}

	return [2]int{s, len(src)}
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
