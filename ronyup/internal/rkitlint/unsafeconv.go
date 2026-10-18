package rkitlint

import (
	"go/ast"
	"go/token"
	"go/types"
)

// checkUnsafeCall matches unsafe.String(unsafe.SliceData(b), len(b)) → rkit.B2S(b) and
// unsafe.Slice(unsafe.StringData(s), len(s)) → rkit.S2B(s).
func (c *checker) checkUnsafeCall(fc *fileCtx, call *ast.CallExpr) {
	var inner, fn string

	switch {
	case c.isUnsafe(call.Fun, "String"):
		inner, fn = "SliceData", "B2S"
	case c.isUnsafe(call.Fun, "Slice"):
		inner, fn = "StringData", "S2B"
	default:
		return
	}

	if len(call.Args) != 2 {
		return
	}

	data, ok := call.Args[0].(*ast.CallExpr)
	if !ok || !c.isUnsafe(data.Fun, inner) || len(data.Args) != 1 {
		return
	}

	length, ok := call.Args[1].(*ast.CallExpr)
	if !ok || !c.isBuiltin(length.Fun, "len") || len(length.Args) != 1 {
		return
	}

	arg := data.Args[0]
	if types.ExprString(arg) != types.ExprString(length.Args[0]) {
		return
	}

	c.reportUnsafe(fc, call, fn, arg)
}

// checkUnsafeDeref matches the pre-Go 1.20 header casts *(*string)(unsafe.Pointer(&b)) → rkit.B2S(b)
// and *(*[]byte)(unsafe.Pointer(&s)) → rkit.S2B(s).
func (c *checker) checkUnsafeDeref(fc *fileCtx, star *ast.StarExpr) {
	conv, ok := star.X.(*ast.CallExpr)
	if !ok || len(conv.Args) != 1 {
		return
	}

	ptr, ok := c.typeOf(conv.Fun).(*types.Pointer)
	if !ok || !c.isTypeExpr(conv.Fun) {
		return
	}

	up, ok := conv.Args[0].(*ast.CallExpr)
	if !ok || len(up.Args) != 1 || !c.isUnsafe(up.Fun, "Pointer") {
		return
	}

	addr, ok := up.Args[0].(*ast.UnaryExpr)
	if !ok || addr.Op != token.AND {
		return
	}

	switch {
	case isString(ptr.Elem()) && isByteSlice(c.typeOf(addr.X)):
		c.reportUnsafe(fc, star, "B2S", addr.X)
	case isByteSlice(ptr.Elem()) && isString(c.typeOf(addr.X)):
		c.reportUnsafe(fc, star, "S2B", addr.X)
	}
}

func (c *checker) reportUnsafe(fc *fileCtx, n ast.Node, fn string, arg ast.Expr) {
	what := "zero-copy []byte→string conversion"
	if fn == "S2B" {
		what = "zero-copy string→[]byte conversion"
	}

	c.report(fc, n.Pos(), n.End(), CategoryUnsafeConv,
		"hand-rolled "+what+": use rkit."+fn,
		"Replace with rkit."+fn,
		fc.rkit(fn+"("+fc.text(arg)+")"), arg)
}

// isUnsafe reports whether e refers to unsafe.<name>.
func (c *checker) isUnsafe(e ast.Expr, name string) bool {
	sel, ok := ast.Unparen(e).(*ast.SelectorExpr)
	if !ok {
		return false
	}

	obj := c.pass.TypesInfo.Uses[sel.Sel]

	return obj != nil && obj == types.Unsafe.Scope().Lookup(name)
}

func (c *checker) isTypeExpr(e ast.Expr) bool {
	tv, ok := c.pass.TypesInfo.Types[e]

	return ok && tv.IsType()
}

func isString(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)

	return ok && b.Kind() == types.String
}

func isByteSlice(t types.Type) bool {
	s, ok := t.Underlying().(*types.Slice)
	if !ok {
		return false
	}

	b, ok := s.Elem().Underlying().(*types.Basic)

	return ok && b.Kind() == types.Byte
}
