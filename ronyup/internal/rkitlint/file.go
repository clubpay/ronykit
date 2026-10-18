package rkitlint

import (
	"bytes"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// fileCtx holds per-file state: the source text for building replacements, the lines silenced by
// the ignore directive, and how rkit is (or would be) imported.
type fileCtx struct {
	pass  *analysis.Pass
	file  *ast.File
	tok   *token.File
	src   []byte
	quiet map[int]bool

	// qual is the selector prefix for rkit identifiers ("rkit.", an alias, or "" for a dot-import).
	qual string
	// imported reports whether the file already imports rkit.
	imported bool
	// canFix is false when no fix can be offered safely (unreadable source, or "rkit" is taken).
	canFix bool
	// importNames maps the import paths of this file to the name they are referred to by
	// ("" for a dot-import), for spelling types in generated closures.
	importNames map[string]string
}

func newFileCtx(pass *analysis.Pass, f *ast.File) *fileCtx {
	fc := &fileCtx{
		pass:        pass,
		file:        f,
		tok:         pass.Fset.File(f.FileStart),
		quiet:       make(map[int]bool),
		qual:        "rkit.",
		canFix:      true,
		importNames: make(map[string]string),
	}

	if src, err := pass.ReadFile(fc.tok.Name()); err == nil && len(src) == fc.tok.Size() {
		fc.src = src
	} else {
		fc.canFix = false
	}

	for _, cg := range f.Comments {
		for _, cm := range cg.List {
			if strings.HasPrefix(cm.Text, "//"+ignoreDirective) {
				fc.quiet[fc.tok.Line(cm.Pos())] = true
			}
		}
	}

	for _, imp := range f.Imports {
		if pn := pass.TypesInfo.PkgNameOf(imp); pn != nil {
			name := pn.Name()
			if name == "." {
				name = ""
			}

			if name != "_" {
				fc.importNames[pn.Imported().Path()] = name
			}
		}

		if path, _ := strconv.Unquote(imp.Path.Value); path != RkitPath {
			continue
		}

		fc.imported = true

		switch {
		case imp.Name == nil:
		case imp.Name.Name == ".":
			fc.qual = ""
		case imp.Name.Name == "_":
			fc.imported = false
		default:
			fc.qual = imp.Name.Name + "."
		}
	}

	if !fc.imported && pass.Pkg.Scope().Lookup("rkit") != nil {
		fc.canFix = false
	}

	return fc
}

func (fc *fileCtx) suppressed(pos token.Pos) bool {
	line := fc.tok.Line(pos)

	return fc.quiet[line] || fc.quiet[line-1]
}

// text returns the source of n, or "" when the source is unavailable (fixes are disabled then).
func (fc *fileCtx) text(n ast.Node) string {
	if fc.src == nil {
		return ""
	}

	return string(fc.src[fc.tok.Offset(n.Pos()):fc.tok.Offset(n.End())])
}

// trailingComment returns the comment group that starts on the line n ends on, after n.
func (fc *fileCtx) trailingComment(n ast.Node) *ast.CommentGroup {
	line := fc.tok.Line(n.End())

	for _, cg := range fc.file.Comments {
		if cg.Pos() >= n.End() && fc.tok.Line(cg.Pos()) == line {
			return cg
		}
	}

	return nil
}

func (fc *fileCtx) hasCommentsIn(pos, end token.Pos) bool {
	for _, cg := range fc.file.Comments {
		if cg.Pos() > pos && cg.End() < end {
			return true
		}
	}

	return false
}

// rkit returns the qualified name of an rkit identifier as it must be spelled in this file.
func (fc *fileCtx) rkit(name string) string {
	return fc.qual + name
}

// rkitEdits replaces [pos, end) with repl and adds the rkit import when the file does not have it
// yet. repl must spell rkit identifiers via fc.rkit.
func (fc *fileCtx) rkitEdits(pos, end token.Pos, repl string) ([]analysis.TextEdit, bool) {
	if !fc.canFix {
		return nil, false
	}

	edits := []analysis.TextEdit{{Pos: pos, End: end, NewText: []byte(repl)}}

	if !fc.imported {
		imp, ok := fc.importEdit()
		if !ok {
			return nil, false
		}

		edits = append(edits, imp)
	}

	return edits, true
}

// importEdit adds the rkit import: next to an existing RonyKit import when there is one (same
// group), otherwise as a new last group. gofmt then sorts it within its group.
func (fc *fileCtx) importEdit() (analysis.TextEdit, bool) {
	spec := strconv.Quote(RkitPath)

	var (
		decl    *ast.GenDecl
		sibling *ast.ImportSpec
	)

	for _, d := range fc.file.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok || gd.Tok != token.IMPORT {
			continue
		}

		decl = gd

		for _, s := range gd.Specs {
			imp, ok := s.(*ast.ImportSpec)
			if !ok {
				continue
			}

			if path, _ := strconv.Unquote(imp.Path.Value); strings.HasPrefix(path, ronykitPrefix) {
				decl, sibling = gd, imp
			}
		}
	}

	switch {
	case decl == nil:
		at := fc.file.Name.End()

		return analysis.TextEdit{Pos: at, End: at, NewText: []byte("\n\nimport " + spec)}, true
	case !decl.Lparen.IsValid():
		sep := "\n\n\t"
		if sibling != nil {
			sep = "\n\t"
		}

		return analysis.TextEdit{
			Pos:     decl.Pos(),
			End:     decl.End(),
			NewText: []byte("import (\n\t" + fc.text(decl.Specs[0]) + sep + spec + "\n)"),
		}, true
	case sibling != nil:
		// Insert on the line after the sibling so a trailing comment stays with its import.
		off := fc.tok.Offset(sibling.End())

		nl := bytes.IndexByte(fc.src[off:], '\n')
		if nl < 0 {
			return analysis.TextEdit{}, false
		}

		at := fc.tok.Pos(off + nl + 1)

		return analysis.TextEdit{Pos: at, End: at, NewText: []byte("\t" + spec + "\n")}, true
	default:
		return analysis.TextEdit{
			Pos:     decl.Rparen,
			End:     decl.Rparen,
			NewText: []byte("\n\t" + spec + "\n"),
		}, true
	}
}

// typeExpr spells t as it must be written in this file. ok is false when that is impossible: a
// package the file does not import, or an unexported type of another package.
func (fc *fileCtx) typeExpr(t types.Type) (string, bool) {
	ok := !hasForeignUnexported(t, fc.pass.Pkg, make(map[types.Type]bool))

	s := types.TypeString(t, func(p *types.Package) string {
		if p == fc.pass.Pkg {
			return ""
		}

		name, found := fc.importNames[p.Path()]
		if !found {
			ok = false

			return p.Name()
		}

		return name
	})

	return s, ok
}

// hasForeignUnexported reports whether t mentions a named type, struct field or method that is
// unexported and belongs to a package other than pkg.
func hasForeignUnexported(t types.Type, pkg *types.Package, seen map[types.Type]bool) bool {
	if seen[t] {
		return false
	}

	seen[t] = true

	foreign := func(obj types.Object) bool {
		return obj.Pkg() != nil && obj.Pkg() != pkg && !obj.Exported()
	}

	switch t := t.(type) {
	case *types.Named:
		if foreign(t.Obj()) {
			return true
		}

		for arg := range t.TypeArgs().Types() {
			if hasForeignUnexported(arg, pkg, seen) {
				return true
			}
		}
	case *types.Alias:
		return foreign(t.Obj()) || hasForeignUnexported(types.Unalias(t), pkg, seen)
	case *types.Pointer:
		return hasForeignUnexported(t.Elem(), pkg, seen)
	case *types.Slice:
		return hasForeignUnexported(t.Elem(), pkg, seen)
	case *types.Array:
		return hasForeignUnexported(t.Elem(), pkg, seen)
	case *types.Chan:
		return hasForeignUnexported(t.Elem(), pkg, seen)
	case *types.Map:
		return hasForeignUnexported(t.Key(), pkg, seen) || hasForeignUnexported(t.Elem(), pkg, seen)
	case *types.Signature:
		return hasForeignUnexported(t.Params(), pkg, seen) || hasForeignUnexported(t.Results(), pkg, seen)
	case *types.Tuple:
		for v := range t.Variables() {
			if hasForeignUnexported(v.Type(), pkg, seen) {
				return true
			}
		}
	case *types.Struct:
		for field := range t.Fields() {
			if foreign(field) || hasForeignUnexported(field.Type(), pkg, seen) {
				return true
			}
		}
	case *types.Interface:
		for m := range t.Methods() {
			if foreign(m) {
				return true
			}
		}
	}

	return false
}
