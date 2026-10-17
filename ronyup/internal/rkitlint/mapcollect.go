package rkitlint

import (
	"go/ast"
	"go/token"
	"go/types"
)

// checkMapCollect matches a slice declaration followed by a loop that only appends a map's keys
// (or values) to it:
//
//	out := make([]K, 0, len(m))
//	for k := range m {
//		out = append(out, k)
//	}
//
// → `out := rkit.MapKeysToArray(m)` (rkit.MapToArray for values). A directly following
// `return out` is folded in. `var out []K` is reported without a fix, because rkit returns an
// empty, non-nil slice where the loop leaves nil.
func (c *checker) checkMapCollect(fc *fileCtx, list []ast.Stmt, i int) {
	if i+1 >= len(list) {
		return
	}

	rng, ok := list[i+1].(*ast.RangeStmt)
	if !ok || rng.Tok != token.DEFINE || len(rng.Body.List) != 1 {
		return
	}

	mt, ok := c.typeOf(rng.X).Underlying().(*types.Map)
	if !ok {
		return
	}

	sObj, elemObj := c.appendTarget(rng.Body.List[0])
	if sObj == nil {
		return
	}

	var (
		fn   string
		want types.Type
	)

	switch keyObj := c.pass.TypesInfo.Defs[identOf(rng.Key)]; {
	case keyObj != nil && elemObj == keyObj && isBlank(rng.Value):
		fn, want = "MapKeysToArray", mt.Key()
	case rng.Value != nil && isBlank(rng.Key) && elemObj == c.pass.TypesInfo.Defs[identOf(rng.Value)]:
		fn, want = "MapToArray", mt.Elem()
	default:
		return
	}

	// Collecting into []any (or any other element type) is a conversion, not a duplicate.
	sl, ok := sObj.Type().Underlying().(*types.Slice)
	if !ok || !types.Identical(sl.Elem(), want) {
		return
	}

	declared, fixable := c.emptySliceDecl(list[i], sObj)
	if !declared {
		return
	}

	// rkit returns []E; a named slice type would change the variable's type.
	fixable = fixable && types.Identical(sObj.Type(), types.NewSlice(want))

	msg := "loop collects map keys: use rkit.MapKeysToArray"
	if fn == "MapToArray" {
		msg = "loop collects map values: use rkit.MapToArray"
	}

	if !fixable {
		c.report(fc, list[i].Pos(), rng.End(), CategoryMapCollect, msg, "", "")

		return
	}

	call := fc.rkit(fn + "(" + fc.text(rng.X) + ")")
	end := rng.End()
	repl := sObj.Name() + " := " + call

	// Uses: two in the loop body, plus the return.
	if i+2 < len(list) && c.uses[sObj] == 3 && c.returnsOnly(list[i+2], sObj) {
		repl = "return " + call
		end = list[i+2].End()
	}

	c.report(fc, list[i].Pos(), end, CategoryMapCollect, msg, "Replace with rkit."+fn, repl)
}

// appendTarget matches `s = append(s, x)` and returns the objects of s and x.
func (c *checker) appendTarget(stmt ast.Stmt) (s, x types.Object) {
	as, ok := stmt.(*ast.AssignStmt)
	if !ok || as.Tok != token.ASSIGN || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
		return nil, nil
	}

	call, ok := as.Rhs[0].(*ast.CallExpr)
	if !ok || !c.isBuiltin(call.Fun, "append") || len(call.Args) != 2 || call.Ellipsis.IsValid() {
		return nil, nil
	}

	s = c.objOf(as.Lhs[0])
	if s == nil || c.objOf(call.Args[0]) != s {
		return nil, nil
	}

	x = c.objOf(call.Args[1])
	if x == nil {
		return nil, nil
	}

	return s, x
}

// emptySliceDecl reports whether stmt declares sObj as an empty slice. fixable is true for
// `s := make(T, 0[, cap])` and `s := T{}`, false for `var s T` (nil, not empty).
func (c *checker) emptySliceDecl(stmt ast.Stmt, sObj types.Object) (declared, fixable bool) {
	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		if stmt.Tok != token.DEFINE || len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 ||
			c.pass.TypesInfo.Defs[identOf(stmt.Lhs[0])] != sObj {
			return false, false
		}

		switch rhs := stmt.Rhs[0].(type) {
		case *ast.CompositeLit:
			return len(rhs.Elts) == 0, len(rhs.Elts) == 0
		case *ast.CallExpr:
			ok := c.isBuiltin(rhs.Fun, "make") && len(rhs.Args) >= 2 && isZeroLit(rhs.Args[1]) &&
				(len(rhs.Args) == 2 || c.isPure(rhs.Args[2]))

			return ok, ok
		}
	case *ast.DeclStmt:
		gd, ok := stmt.Decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR || len(gd.Specs) != 1 {
			return false, false
		}

		vs, ok := gd.Specs[0].(*ast.ValueSpec)
		if ok && len(vs.Names) == 1 && len(vs.Values) == 0 && c.pass.TypesInfo.Defs[vs.Names[0]] == sObj {
			return true, false
		}
	}

	return false, false
}

// isPure reports whether dropping e cannot drop a side effect: identifiers, literals, selectors,
// and len/cap of those.
func (c *checker) isPure(e ast.Expr) bool {
	switch e := ast.Unparen(e).(type) {
	case *ast.Ident, *ast.BasicLit:
		return true
	case *ast.SelectorExpr:
		return c.isPure(e.X)
	case *ast.CallExpr:
		return (c.isBuiltin(e.Fun, "len") || c.isBuiltin(e.Fun, "cap")) && len(e.Args) == 1 &&
			c.isPure(e.Args[0])
	}

	return false
}

func isZeroLit(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)

	return ok && lit.Kind == token.INT && lit.Value == "0"
}

func identOf(e ast.Expr) *ast.Ident {
	id, _ := e.(*ast.Ident)

	return id
}
