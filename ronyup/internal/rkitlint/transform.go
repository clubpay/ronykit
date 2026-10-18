package rkitlint

import (
	"go/ast"
	"go/token"
	"go/types"
)

// checkTransform matches a slice declaration followed by a loop over another slice that only
// maps or filters its elements:
//
//	out := make([]D, 0, len(src))      out := make([]D, len(src))     out := []S{}
//	for _, x := range src {            for i, x := range src {        for _, x := range src {
//		out = append(out, f(x))            out[i] = f(x)                  if keep(x) {
//	}                                  }                                      out = append(out, x)
//	                                                                      }
//	                                                                  }
//
// → `out := rkit.Map(src, f)` / `out := rkit.Filter(src, func(x S) bool { return keep(x) })`.
// A directly following `return out` is folded in.
func (c *checker) checkTransform(fc *fileCtx, list []ast.Stmt, i int) {
	if i+1 >= len(list) {
		return
	}

	rng, ok := list[i+1].(*ast.RangeStmt)
	if !ok || rng.Tok != token.DEFINE || len(rng.Body.List) != 1 {
		return
	}

	src, ok := c.typeOf(rng.X).Underlying().(*types.Slice)
	if !ok {
		return
	}

	x := c.pass.TypesInfo.Defs[identOf(rng.Value)]
	if x == nil {
		return
	}

	if c.checkMapLoop(fc, list, i, rng, src, x) {
		return
	}

	c.checkFilterLoop(fc, list, i, rng, src, x)
}

func (c *checker) checkMapLoop(
	fc *fileCtx, list []ast.Stmt, i int, rng *ast.RangeStmt, src *types.Slice, x types.Object,
) bool {
	as, ok := rng.Body.List[0].(*ast.AssignStmt)
	if !ok || as.Tok != token.ASSIGN || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
		return false
	}

	var (
		out      types.Object
		expr     ast.Expr
		outUses  int
		declared bool
		fixable  bool
	)

	switch lhs := as.Lhs[0].(type) {
	case *ast.Ident: // out = append(out, expr)
		call, isCall := as.Rhs[0].(*ast.CallExpr)
		if !isCall || !c.isBuiltin(call.Fun, "append") || len(call.Args) != 2 || call.Ellipsis.IsValid() ||
			!isBlank(rng.Key) {
			return false
		}

		out = c.objOf(lhs)
		if out == nil || c.objOf(call.Args[0]) != out {
			return false
		}

		expr, outUses = call.Args[1], 2
		declared, fixable = c.emptySliceDecl(list[i], out)
	case *ast.IndexExpr: // out[i] = expr, with out := make([]D, len(src))
		key := c.pass.TypesInfo.Defs[identOf(rng.Key)]
		if key == nil || c.objOf(lhs.Index) != key || c.uses[key] != 1 {
			return false
		}

		out = c.objOf(lhs.X)
		if out == nil {
			return false
		}

		expr, outUses = as.Rhs[0], 1
		declared = c.lenOfDecl(list[i], out, rng.X)
		fixable = declared
	default:
		return false
	}

	// Appending x itself is a copy, not a map.
	if !declared || c.references(expr, out) || c.objOf(expr) == x {
		return false
	}

	outSlice, ok := out.Type().Underlying().(*types.Slice)
	if !ok {
		return false
	}

	foldReturn := i+2 < len(list) && c.uses[out] == outUses+1 && c.returnsOnly(list[i+2], out)

	fn, ok := c.funcValue(fc, expr, x, src.Elem(), outSlice.Elem())
	if !ok {
		fn, ok = c.closure(fc, x, src.Elem(), outSlice.Elem(), expr)
	}

	fixable = fixable && ok && types.Identical(out.Type(), types.NewSlice(outSlice.Elem()))
	c.reportTransform(fc, list, i, rng, out, foldReturn, fixable, "Map", "loop maps one slice to another", fn)

	return true
}

func (c *checker) checkFilterLoop(
	fc *fileCtx, list []ast.Stmt, i int, rng *ast.RangeStmt, src *types.Slice, x types.Object,
) {
	ifs, ok := rng.Body.List[0].(*ast.IfStmt)
	if !ok || ifs.Init != nil || ifs.Else != nil || len(ifs.Body.List) != 1 || !isBlank(rng.Key) {
		return
	}

	out, elem := c.appendTarget(ifs.Body.List[0])
	if out == nil || elem != x || c.references(ifs.Cond, out) {
		return
	}

	declared, fixable := c.emptySliceDecl(list[i], out)
	if !declared {
		return
	}

	outSlice, ok := out.Type().Underlying().(*types.Slice)
	if !ok || !types.Identical(outSlice.Elem(), src.Elem()) {
		return
	}

	foldReturn := i+2 < len(list) && c.uses[out] == 3 && c.returnsOnly(list[i+2], out)

	boolType := types.Typ[types.Bool]

	fn, ok := c.funcValue(fc, ifs.Cond, x, src.Elem(), boolType)
	if !ok {
		fn, ok = c.closure(fc, x, src.Elem(), boolType, ifs.Cond)
	}

	fixable = fixable && ok && types.Identical(out.Type(), types.NewSlice(src.Elem()))
	c.reportTransform(fc, list, i, rng, out, foldReturn, fixable, "Filter", "loop filters a slice", fn)
}

func (c *checker) reportTransform(
	fc *fileCtx, list []ast.Stmt, i int, rng *ast.RangeStmt, out types.Object,
	foldReturn, fixable bool, helper, what, fn string,
) {
	msg := what + ": use rkit." + helper
	if !fixable {
		c.report(fc, list[i].Pos(), rng.End(), CategoryTransform, msg, "", "")

		return
	}

	call := fc.rkit(helper + "(" + fc.text(rng.X) + ", " + fn + ")")
	end := rng.End()
	repl := out.Name() + " := " + call

	if foldReturn {
		repl = "return " + call
		end = list[i+2].End()
	}

	c.report(fc, list[i].Pos(), end, CategoryTransform, msg, "Replace with rkit."+helper, repl)
}

// funcValue returns f when expr is exactly `f(x)` and f can be passed as a func(S) R as is.
func (c *checker) funcValue(fc *fileCtx, expr ast.Expr, x types.Object, s, r types.Type) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || call.Ellipsis.IsValid() || c.objOf(call.Args[0]) != x {
		return "", false
	}

	var obj types.Object

	switch fun := call.Fun.(type) {
	case *ast.Ident:
		obj = c.pass.TypesInfo.Uses[fun]
	case *ast.SelectorExpr:
		// pkg.Func, or a method value on a plain variable (evaluated once instead of per element).
		if !isPlainRef(fun.X) {
			return "", false
		}

		obj = c.pass.TypesInfo.Uses[fun.Sel]
	default:
		return "", false
	}

	switch obj := obj.(type) {
	case *types.Func:
		if sig, isSig := obj.Type().(*types.Signature); !isSig || sig.TypeParams().Len() > 0 {
			return "", false // a generic func would need explicit instantiation
		}
	case *types.Var:
	default:
		return "", false // conversions and builtins are not func values
	}

	sig, ok := c.typeOf(call.Fun).(*types.Signature)
	if !ok || sig.TypeParams().Len() > 0 || sig.Variadic() || sig.Params().Len() != 1 ||
		sig.Results().Len() != 1 {
		return "", false
	}

	if !types.Identical(sig.Params().At(0).Type(), s) || !types.Identical(sig.Results().At(0).Type(), r) {
		return "", false
	}

	return fc.text(call.Fun), true
}

// closure builds `func(x S) R { return expr }`.
func (c *checker) closure(fc *fileCtx, x types.Object, s, r types.Type, expr ast.Expr) (string, bool) {
	st, ok := fc.typeExpr(s)
	if !ok {
		return "", false
	}

	rt, ok := fc.typeExpr(r)
	if !ok {
		return "", false
	}

	return "func(" + x.Name() + " " + st + ") " + rt + " { return " + fc.text(expr) + " }", true
}

// lenOfDecl reports whether stmt is `out := make(T, len(src)[, cap])`.
func (c *checker) lenOfDecl(stmt ast.Stmt, out types.Object, src ast.Expr) bool {
	as, ok := stmt.(*ast.AssignStmt)
	if !ok || as.Tok != token.DEFINE || len(as.Lhs) != 1 || len(as.Rhs) != 1 ||
		c.pass.TypesInfo.Defs[identOf(as.Lhs[0])] != out {
		return false
	}

	mk, ok := as.Rhs[0].(*ast.CallExpr)
	if !ok || !c.isBuiltin(mk.Fun, "make") || len(mk.Args) < 2 ||
		(len(mk.Args) == 3 && !c.isPure(mk.Args[2])) {
		return false
	}

	ln, ok := mk.Args[1].(*ast.CallExpr)

	return ok && c.isBuiltin(ln.Fun, "len") && len(ln.Args) == 1 && sameExpr(ln.Args[0], src)
}

// references reports whether n mentions obj.
func (c *checker) references(n ast.Node, obj types.Object) bool {
	found := false

	ast.Inspect(n, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && c.pass.TypesInfo.Uses[id] == obj {
			found = true
		}

		return !found
	})

	return found
}
