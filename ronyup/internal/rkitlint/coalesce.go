package rkitlint

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// checkCoalesceDefault matches a run of default assignments to the same variable,
//
//	if x == "" {
//		x = y
//	}
//	if x == "" {
//		x = z
//	}
//
// → `x = rkit.Coalesce(x, y, z)`. Every fallback must be free of side effects, because Coalesce
// evaluates all its arguments.
func (c *checker) checkCoalesceDefault(fc *fileCtx, list []ast.Stmt, i int) {
	target, fallback := c.defaultAssign(list[i])
	if target == nil {
		return
	}

	// Report a run once, from its first statement.
	if i > 0 {
		if prev, _ := c.defaultAssign(list[i-1]); prev != nil && sameExpr(prev, target) {
			return
		}
	}

	args := []string{fc.text(target), fc.text(fallback)}
	keep := []ast.Node{target, fallback}
	end := list[i].End()

	for j := i + 1; j < len(list); j++ {
		next, fb := c.defaultAssign(list[j])
		if next == nil || !sameExpr(next, target) {
			break
		}

		args = append(args, fc.text(fb))
		keep = append(keep, fb)
		end = list[j].End()
	}

	c.report(fc, list[i].Pos(), end, CategoryCoalesce,
		"default value assigned with if: use rkit.Coalesce",
		"Replace with rkit.Coalesce",
		fc.text(target)+" = "+fc.rkit("Coalesce("+strings.Join(args, ", ")+")"), keep...)
}

// defaultAssign matches `if x == <zero> { x = y }` and returns x and y.
func (c *checker) defaultAssign(stmt ast.Stmt) (target, fallback ast.Expr) {
	ifs, ok := stmt.(*ast.IfStmt)
	if !ok || ifs.Init != nil || ifs.Else != nil || len(ifs.Body.List) != 1 {
		return nil, nil
	}

	cond, ok := ifs.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.EQL {
		return nil, nil
	}

	as, ok := ifs.Body.List[0].(*ast.AssignStmt)
	if !ok || as.Tok != token.ASSIGN || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
		return nil, nil
	}

	x := cond.X
	if !c.isZero(cond.Y) || !isPlainRef(x) || !sameExpr(x, as.Lhs[0]) || !c.coalescable(x, as.Rhs[0]) {
		return nil, nil
	}

	return x, as.Rhs[0]
}

// checkCoalesceReturn matches a run of early returns ending the block,
//
//	if a != "" {
//		return a
//	}
//	if b != "" {
//		return b
//	}
//	return "default"
//
// → `return rkit.Coalesce(a, b, "default")`.
func (c *checker) checkCoalesceReturn(fc *fileCtx, list []ast.Stmt, i int) {
	if c.returnIfNonZero(list[i]) == nil {
		return
	}

	if i > 0 && c.returnIfNonZero(list[i-1]) != nil {
		return
	}

	var (
		vals []ast.Expr
		j    = i
	)

	for ; j < len(list); j++ {
		v := c.returnIfNonZero(list[j])
		if v == nil {
			break
		}

		vals = append(vals, v)
	}

	// The run must end the block with `return w`.
	if j != len(list)-1 {
		return
	}

	ret, ok := list[j].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return
	}

	last := ret.Results[0]

	args := make([]string, 0, len(vals)+1)
	for _, v := range vals {
		if !types.Identical(c.typeOf(v), c.typeOf(vals[0])) {
			return
		}

		args = append(args, fc.text(v))
	}

	if !c.coalescable(vals[0], last) {
		return
	}

	args = append(args, fc.text(last))

	keep := make([]ast.Node, 0, len(vals)+1)
	for _, v := range vals {
		keep = append(keep, v)
	}

	keep = append(keep, last)

	c.report(fc, list[i].Pos(), ret.End(), CategoryCoalesce,
		"chain of non-zero checks returning the first value: use rkit.Coalesce",
		"Replace with rkit.Coalesce",
		"return "+fc.rkit("Coalesce("+strings.Join(args, ", ")+")"), keep...)
}

// returnIfNonZero matches `if v != <zero> { return v }` and returns v.
func (c *checker) returnIfNonZero(stmt ast.Stmt) ast.Expr {
	ifs, ok := stmt.(*ast.IfStmt)
	if !ok || ifs.Init != nil || ifs.Else != nil || len(ifs.Body.List) != 1 {
		return nil
	}

	cond, ok := ifs.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.NEQ || !c.isZero(cond.Y) || !isPlainRef(cond.X) {
		return nil
	}

	ret, ok := ifs.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 || !sameExpr(ret.Results[0], cond.X) {
		return nil
	}

	if t := c.typeOf(cond.X); t == nil || !comparableValue(t) {
		return nil
	}

	return cond.X
}

// coalescable reports whether rkit.Coalesce(x, y) compiles and behaves like the if: x has a
// comparable, non-interface type, and y is side-effect free and has the same type (or is an
// untyped constant).
func (c *checker) coalescable(x, y ast.Expr) bool {
	xt := c.typeOf(x)
	if xt == nil || !comparableValue(xt) {
		return false
	}

	tv, ok := c.pass.TypesInfo.Types[y]
	if !ok || (tv.Value == nil && !c.isPure(y)) {
		return false
	}

	if b, isBasic := tv.Type.(*types.Basic); isBasic && b.Info()&types.IsUntyped != 0 {
		return b.Kind() != types.UntypedNil
	}

	return types.Identical(tv.Type, xt)
}

// comparableValue: Coalesce needs `comparable`; interfaces satisfy it but can panic at run time.
func comparableValue(t types.Type) bool {
	return types.Comparable(t) && !types.IsInterface(t)
}

// isPlainRef reports whether e is an identifier or a chain of field selections on one.
func isPlainRef(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return isPlainRef(e.X)
	}

	return false
}

func sameExpr(a, b ast.Expr) bool {
	return types.ExprString(a) == types.ExprString(b)
}
