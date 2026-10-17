// Package rkitlint reports hand-written Go code that duplicates an x/rkit helper.
//
// The rules are deliberately narrow so that every report is a real duplicate:
//
//   - must:       `v, err := f(); if err != nil { panic(err) }` → rkit.Must / rkit.Assert
//   - mapcollect: a loop that only appends a map's keys or values → rkit.MapKeysToArray / rkit.MapToArray
//   - unsafeconv: hand-rolled zero-copy []byte↔string conversions → rkit.B2S / rkit.S2B
//   - helper:     unexported functions whose body re-implements rkit.Must, PtrVal, Contains or Coalesce
//
// Most reports carry a suggested fix that rewrites the code and adds the rkit import.
// A report is suppressed by a `//rkitcheck:ignore <reason>` comment on its line or the line above.
// (Not //nolint: golangci-lint parses those and warns about a linter it does not know.)
package rkitlint

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const (
	// RkitPath is the import path of the helper package this analyzer steers code towards.
	RkitPath = "github.com/clubpay/ronykit/x/rkit"

	ignoreDirective = "rkitcheck:ignore"
	ronykitPrefix   = "github.com/clubpay/ronykit/"
)

// Report categories.
const (
	CategoryMust       = "must"
	CategoryMapCollect = "mapcollect"
	CategoryUnsafeConv = "unsafeconv"
	CategoryHelper     = "helper"
)

var Analyzer = &analysis.Analyzer{
	Name: "rkitcheck",
	Doc:  "reports hand-written code that duplicates an x/rkit helper",
	URL:  "https://pkg.go.dev/" + RkitPath,
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Path() == RkitPath || strings.HasPrefix(pass.Pkg.Path(), RkitPath+"/") {
		return nil, nil //nolint:nilnil
	}

	c := &checker{
		pass: pass,
		uses: make(map[types.Object]int),
	}
	for _, obj := range pass.TypesInfo.Uses {
		c.uses[obj]++
	}

	for _, f := range pass.Files {
		if ast.IsGenerated(f) {
			continue
		}

		c.checkFile(newFileCtx(pass, f))
	}

	return nil, nil //nolint:nilnil
}

type checker struct {
	pass *analysis.Pass
	uses map[types.Object]int
}

func (c *checker) checkFile(fc *fileCtx) {
	ast.Inspect(fc.file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BlockStmt:
			c.checkStmts(fc, n.List)
		case *ast.CaseClause:
			c.checkStmts(fc, n.Body)
		case *ast.CommClause:
			c.checkStmts(fc, n.Body)
		case *ast.CallExpr:
			c.checkUnsafeCall(fc, n)
		case *ast.StarExpr:
			c.checkUnsafeDeref(fc, n)
		case *ast.FuncDecl:
			c.checkHelper(fc, n)
		}

		return true
	})
}

func (c *checker) checkStmts(fc *fileCtx, list []ast.Stmt) {
	for i := range list {
		c.checkMust(fc, list, i)
		c.checkAssertIfInit(fc, list[i])
		c.checkMapCollect(fc, list, i)
	}
}

// report emits a diagnostic for [pos, end). When repl is non-empty it attaches a fix that replaces
// that range with repl and imports rkit if needed.
func (c *checker) report(fc *fileCtx, pos, end token.Pos, category, msg, fixMsg, repl string) {
	if fc.suppressed(pos) {
		return
	}

	d := analysis.Diagnostic{
		Pos:      pos,
		End:      end,
		Category: category,
		Message:  msg,
	}

	if repl != "" && !fc.hasCommentsIn(pos, end) {
		if edits, ok := fc.rkitEdits(pos, end, repl); ok {
			d.SuggestedFixes = []analysis.SuggestedFix{{Message: fixMsg, TextEdits: edits}}
		}
	}

	c.pass.Report(d)
}

func (c *checker) typeOf(e ast.Expr) types.Type {
	return c.pass.TypesInfo.TypeOf(e)
}

// objOf returns the object an identifier expression defines or uses, or nil.
func (c *checker) objOf(e ast.Expr) types.Object {
	id, ok := ast.Unparen(e).(*ast.Ident)
	if !ok {
		return nil
	}

	return c.pass.TypesInfo.ObjectOf(id)
}

func (c *checker) isBuiltin(e ast.Expr, name string) bool {
	b, ok := c.objOf(e).(*types.Builtin)

	return ok && b.Name() == name && b.Parent() == types.Universe
}

func (c *checker) isNil(e ast.Expr) bool {
	tv, ok := c.pass.TypesInfo.Types[e]

	return ok && tv.IsNil()
}

func isBlank(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)

	return e == nil || (ok && id.Name == "_")
}

var errorType = types.Universe.Lookup("error").Type()

func isError(t types.Type) bool {
	return t != nil && types.Identical(t, errorType)
}
