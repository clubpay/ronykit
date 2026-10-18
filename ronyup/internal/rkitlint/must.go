package rkitlint

import (
	"go/ast"
	"go/token"
	"go/types"
)

// checkMust matches `v, err := f()` or `err := f()` directly followed by
//
//	if err != nil {
//		panic(err)
//	}
//
// where err is declared by the assignment and used nowhere else. The pair becomes
// `v := rkit.Must(f())` or `rkit.Assert(f())`; a directly following `return v` is folded in.
func (c *checker) checkMust(fc *fileCtx, list []ast.Stmt, i int) {
	if i+1 >= len(list) {
		return
	}

	as, ok := list[i].(*ast.AssignStmt)
	if !ok || as.Tok != token.DEFINE || len(as.Rhs) != 1 || len(as.Lhs) > 2 {
		return
	}

	call, ok := as.Rhs[0].(*ast.CallExpr)
	if !ok {
		return
	}

	ifs, ok := list[i+1].(*ast.IfStmt)
	if !ok || ifs.Init != nil {
		return
	}

	errID, ok := as.Lhs[len(as.Lhs)-1].(*ast.Ident)
	if !ok {
		return
	}

	errObj := c.pass.TypesInfo.Defs[errID]
	if errObj == nil || !isError(errObj.Type()) || c.uses[errObj] != 2 || !c.isPanicOnErr(ifs, errObj) {
		return
	}

	callText := fc.text(call)

	if len(as.Lhs) == 1 {
		if !isError(c.typeOf(call)) {
			return
		}

		c.report(fc, as.Pos(), ifs.End(), CategoryMust,
			"if err != nil { panic(err) } after a call: use rkit.Assert",
			"Replace with rkit.Assert",
			fc.rkit("Assert("+callText+")"), call)

		return
	}

	res, ok := c.typeOf(call).(*types.Tuple)
	if !ok || res.Len() != 2 || !isError(res.At(1).Type()) {
		return
	}

	v, ok := as.Lhs[0].(*ast.Ident)
	if !ok {
		return
	}

	must := fc.rkit("Must(" + callText + ")")
	end := ifs.End()

	var repl string

	vObj := c.pass.TypesInfo.Defs[v]

	switch {
	case v.Name == "_":
		repl = must
	case vObj != nil && i+2 < len(list) && c.uses[vObj] == 1 && c.returnsOnly(list[i+2], vObj):
		repl = "return " + must
		end = list[i+2].End()
	case vObj != nil:
		repl = v.Name + " := " + must
	default: // v already existed; only err was declared here.
		repl = v.Name + " = " + must
	}

	c.report(fc, as.Pos(), end, CategoryMust,
		"if err != nil { panic(err) } after a call: use rkit.Must",
		"Replace with rkit.Must",
		repl, call)
}

// checkAssertIfInit matches `if err := f(); err != nil { panic(err) }` → `rkit.Assert(f())`.
func (c *checker) checkAssertIfInit(fc *fileCtx, stmt ast.Stmt) {
	ifs, ok := stmt.(*ast.IfStmt)
	if !ok || ifs.Init == nil {
		return
	}

	as, ok := ifs.Init.(*ast.AssignStmt)
	if !ok || as.Tok != token.DEFINE || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
		return
	}

	call, ok := as.Rhs[0].(*ast.CallExpr)
	if !ok || !isError(c.typeOf(call)) {
		return
	}

	errObj := c.objOf(as.Lhs[0])
	if errObj == nil || c.uses[errObj] != 2 || !c.isPanicOnErr(ifs, errObj) {
		return
	}

	c.report(fc, ifs.Pos(), ifs.End(), CategoryMust,
		"if err := f(); err != nil { panic(err) }: use rkit.Assert",
		"Replace with rkit.Assert",
		fc.rkit("Assert("+fc.text(call)+")"), call)
}

// isPanicOnErr reports whether ifs is exactly `if err != nil { panic(err) }` (init aside, no else).
func (c *checker) isPanicOnErr(ifs *ast.IfStmt, errObj types.Object) bool {
	if ifs.Else != nil || len(ifs.Body.List) != 1 {
		return false
	}

	cond, ok := ifs.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.NEQ {
		return false
	}

	x, y := cond.X, cond.Y
	if c.isNil(x) {
		x, y = y, x
	}

	if c.objOf(x) != errObj || !c.isNil(y) {
		return false
	}

	es, ok := ifs.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}

	call, ok := es.X.(*ast.CallExpr)

	return ok && c.isBuiltin(call.Fun, "panic") && len(call.Args) == 1 && c.objOf(call.Args[0]) == errObj
}

// returnsOnly reports whether stmt is `return v` for the given object.
func (c *checker) returnsOnly(stmt ast.Stmt, obj types.Object) bool {
	ret, ok := stmt.(*ast.ReturnStmt)

	return ok && len(ret.Results) == 1 && c.objOf(ret.Results[0]) == obj
}
