package rkitlint

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
)

// checkHelper reports unexported functions whose whole body re-implements an rkit helper. There is
// no automatic fix: deleting the helper means updating its callers.
func (c *checker) checkHelper(fc *fileCtx, fd *ast.FuncDecl) {
	if fd.Recv != nil || fd.Body == nil || fd.Name.IsExported() {
		return
	}

	fn, ok := c.pass.TypesInfo.Defs[fd.Name].(*types.Func)
	if !ok {
		return
	}

	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.Results().Len() != 1 {
		return
	}

	var dup string

	switch {
	case c.isMustHelper(sig, fd.Body.List):
		dup = "Must"
	case c.isPtrValHelper(sig, fd.Body.List):
		dup = "PtrVal"
	case c.isContainsHelper(sig, fd.Body.List):
		dup = "Contains"
	case c.isCoalesceHelper(sig, fd.Body.List):
		dup = "Coalesce"
	default:
		return
	}

	c.report(fc, fd.Name.Pos(), fd.Name.End(), CategoryHelper,
		"func "+fd.Name.Name+" duplicates rkit."+dup+"; delete it and call rkit."+dup, "", "")
}

// isMustHelper: func(v T, err error) T { if err != nil { panic(err) }; return v }.
func (c *checker) isMustHelper(sig *types.Signature, body []ast.Stmt) bool {
	params := sig.Params()
	if params.Len() != 2 || !isError(params.At(1).Type()) ||
		!types.Identical(params.At(0).Type(), sig.Results().At(0).Type()) || len(body) != 2 {
		return false
	}

	ifs, ok := body[0].(*ast.IfStmt)

	return ok && ifs.Init == nil && c.isPanicOnErr(ifs, params.At(1)) && c.returnsOnly(body[1], params.At(0))
}

// isPtrValHelper: func(p *T) T { if p == nil { return <zero> }; return *p }.
func (c *checker) isPtrValHelper(sig *types.Signature, body []ast.Stmt) bool {
	params := sig.Params()
	if params.Len() != 1 || len(body) != 2 {
		return false
	}

	ptr, ok := params.At(0).Type().(*types.Pointer)
	if !ok || !types.Identical(ptr.Elem(), sig.Results().At(0).Type()) {
		return false
	}

	ifs, ok := body[0].(*ast.IfStmt)
	if !ok || ifs.Init != nil || ifs.Else != nil || !c.isNilCheck(ifs.Cond, params.At(0), token.EQL) ||
		!c.returnsZero(ifs.Body.List) {
		return false
	}

	ret, ok := body[1].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}

	deref, ok := ret.Results[0].(*ast.StarExpr)

	return ok && c.objOf(deref.X) == params.At(0)
}

// isContainsHelper: func(s []T, v T) bool { for _, x := range s { if x == v { return true } }; return false
// }.
func (c *checker) isContainsHelper(sig *types.Signature, body []ast.Stmt) bool {
	params := sig.Params()
	if params.Len() != 2 || len(body) != 2 || !isBool(sig.Results().At(0).Type()) {
		return false
	}

	sl, ok := params.At(0).Type().(*types.Slice)
	if !ok || !types.Identical(sl.Elem(), params.At(1).Type()) {
		return false
	}

	x, ifs := c.rangeIf(body[0], params.At(0))
	if ifs == nil || !c.isCompare(ifs.Cond, token.EQL, x, params.At(1)) ||
		!c.returnsBool(ifs.Body.List, true) {
		return false
	}

	return c.returnsBool(body[1:], false)
}

// isCoalesceHelper: func(vals ...T) T { for _, v := range vals { if v != <zero> { return v } }; return <zero>
// },
// optionally with a leading `var zero T` used as <zero>.
func (c *checker) isCoalesceHelper(sig *types.Signature, body []ast.Stmt) bool {
	params := sig.Params()
	if !sig.Variadic() || params.Len() != 1 {
		return false
	}

	var zeroVar types.Object

	if len(body) == 3 {
		zeroVar = c.zeroVarDecl(body[0])
		if zeroVar == nil {
			return false
		}

		body = body[1:]
	}

	if len(body) != 2 {
		return false
	}

	isZero := func(e ast.Expr) bool {
		return c.isZero(e) || (zeroVar != nil && c.objOf(e) == zeroVar)
	}

	x, ifs := c.rangeIf(body[0], params.At(0))
	if ifs == nil {
		return false
	}

	cond, ok := ifs.Cond.(*ast.BinaryExpr)
	if !ok || cond.Op != token.NEQ || c.objOf(cond.X) != x || !isZero(cond.Y) {
		return false
	}

	if len(ifs.Body.List) != 1 || !c.returnsOnly(ifs.Body.List[0], x) {
		return false
	}

	ret, ok := body[1].(*ast.ReturnStmt)

	return ok && len(ret.Results) == 1 && isZero(ret.Results[0])
}

// rangeIf matches `for _, x := range s { if cond { ... } }` and returns x and the if statement.
func (c *checker) rangeIf(stmt ast.Stmt, s types.Object) (types.Object, *ast.IfStmt) {
	rng, ok := stmt.(*ast.RangeStmt)
	if !ok || rng.Tok != token.DEFINE || !isBlank(rng.Key) || rng.Value == nil ||
		c.objOf(rng.X) != s || len(rng.Body.List) != 1 {
		return nil, nil
	}

	ifs, ok := rng.Body.List[0].(*ast.IfStmt)
	if !ok || ifs.Init != nil || ifs.Else != nil {
		return nil, nil
	}

	x := c.pass.TypesInfo.Defs[identOf(rng.Value)]
	if x == nil {
		return nil, nil
	}

	return x, ifs
}

func (c *checker) isNilCheck(cond ast.Expr, obj types.Object, op token.Token) bool {
	be, ok := cond.(*ast.BinaryExpr)
	if !ok || be.Op != op {
		return false
	}

	return (c.objOf(be.X) == obj && c.isNil(be.Y)) || (c.isNil(be.X) && c.objOf(be.Y) == obj)
}

func (c *checker) isCompare(cond ast.Expr, op token.Token, a, b types.Object) bool {
	be, ok := cond.(*ast.BinaryExpr)
	if !ok || be.Op != op {
		return false
	}

	x, y := c.objOf(be.X), c.objOf(be.Y)

	return (x == a && y == b) || (x == b && y == a)
}

// returnsZero matches `return <zero value>` or `var z T; return z`.
func (c *checker) returnsZero(list []ast.Stmt) bool {
	switch len(list) {
	case 1:
		ret, ok := list[0].(*ast.ReturnStmt)

		return ok && len(ret.Results) == 1 && c.isZero(ret.Results[0])
	case 2:
		z := c.zeroVarDecl(list[0])

		return z != nil && c.returnsOnly(list[1], z)
	}

	return false
}

// zeroVarDecl matches `var z T` (no initializer) and returns z.
func (c *checker) zeroVarDecl(stmt ast.Stmt) types.Object {
	ds, ok := stmt.(*ast.DeclStmt)
	if !ok {
		return nil
	}

	gd, ok := ds.Decl.(*ast.GenDecl)
	if !ok || gd.Tok != token.VAR || len(gd.Specs) != 1 {
		return nil
	}

	vs, ok := gd.Specs[0].(*ast.ValueSpec)
	if !ok || len(vs.Names) != 1 || len(vs.Values) != 0 {
		return nil
	}

	return c.pass.TypesInfo.Defs[vs.Names[0]]
}

func (c *checker) returnsBool(list []ast.Stmt, want bool) bool {
	if len(list) != 1 {
		return false
	}

	ret, ok := list[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}

	tv, ok := c.pass.TypesInfo.Types[ret.Results[0]]

	return ok && tv.Value != nil && tv.Value.Kind() == constant.Bool && constant.BoolVal(tv.Value) == want
}

// isZero reports whether e is a zero-value literal: nil, a zero constant, or an empty composite literal.
func (c *checker) isZero(e ast.Expr) bool {
	if c.isNil(e) {
		return true
	}

	if lit, ok := ast.Unparen(e).(*ast.CompositeLit); ok {
		switch c.typeOf(lit).Underlying().(type) {
		case *types.Struct, *types.Array:
			return len(lit.Elts) == 0
		}

		return false // T{} for a slice or map is empty, not nil
	}

	tv, ok := c.pass.TypesInfo.Types[e]
	if !ok || tv.Value == nil {
		return false
	}

	switch tv.Value.Kind() {
	case constant.Bool:
		return !constant.BoolVal(tv.Value)
	case constant.String:
		return constant.StringVal(tv.Value) == ""
	case constant.Int, constant.Float, constant.Complex:
		return constant.Sign(tv.Value) == 0
	}

	return false
}

func isBool(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)

	return ok && b.Kind() == types.Bool
}
