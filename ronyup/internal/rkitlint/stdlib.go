package rkitlint

import (
	"go/ast"
	"go/constant"
	"go/types"
)

// checkStdlibCall matches standard-library calls that rkit covers (the workspace convention is to
// use rkit whenever it has the helper):
//
//	slices.Contains(s, v)        → rkit.Contains(s, v)
//	cmp.Or(a, b)                 → rkit.Coalesce(a, b)
//	strconv.Itoa(x)              → rkit.IntToStr(x)
//	strconv.FormatInt(x, 10)     → rkit.Int64ToStr(x)
//	strconv.FormatUint(x, 10)    → rkit.UInt64ToStr(x)
//	fmt.Sprintf("%d", x)         → rkit.IntToStr(x) (Int32ToStr, Int64ToStr, UIntToStr, ... by type)
//	slices.Collect(maps.Keys(m)) → rkit.MapKeysToArray(m) (reported only: rkit returns [] where
//	                               Collect returns nil; same for maps.Values / MapToArray)
func (c *checker) checkStdlibCall(fc *fileCtx, call *ast.CallExpr) {
	fn := c.callee(call)
	if fn == nil || call.Ellipsis.IsValid() {
		return
	}

	args := call.Args

	switch fn.FullName() {
	case "slices.Contains":
		c.reportStdlibFun(fc, call, "slices.Contains", "Contains")
	case "cmp.Or":
		c.reportStdlibFun(fc, call, "cmp.Or", "Coalesce")
	case "strconv.Itoa":
		c.reportStdlibCall(fc, call, "strconv.Itoa", "IntToStr", args[0])
	case "strconv.FormatInt":
		if c.isIntConst(args[1], 10) {
			c.reportStdlibCall(fc, call, "strconv.FormatInt(x, 10)", "Int64ToStr", args[0])
		}
	case "strconv.FormatUint":
		if c.isIntConst(args[1], 10) {
			c.reportStdlibCall(fc, call, "strconv.FormatUint(x, 10)", "UInt64ToStr", args[0])
		}
	case "fmt.Sprintf":
		if len(args) != 2 || !c.isStringConst(args[0], "%d") {
			return
		}

		if helper := intToStrHelper(c.typeOf(args[1])); helper != "" {
			c.reportStdlibCall(fc, call, `fmt.Sprintf("%d", x)`, helper, args[1])
		}
	case "slices.Collect":
		inner, ok := args[0].(*ast.CallExpr)
		if !ok {
			return
		}

		const note = " (it returns an empty, non-nil slice)"

		switch f := c.callee(inner); {
		case f == nil:
		case f.FullName() == "maps.Keys":
			msg := "slices.Collect(maps.Keys(m)): use rkit.MapKeysToArray" + note
			c.report(fc, call.Pos(), call.End(), CategoryStdlib, msg, "", "")
		case f.FullName() == "maps.Values":
			msg := "slices.Collect(maps.Values(m)): use rkit.MapToArray" + note
			c.report(fc, call.Pos(), call.End(), CategoryStdlib, msg, "", "")
		}
	}
}

// reportStdlibFun swaps only the function, keeping the arguments as written.
func (c *checker) reportStdlibFun(fc *fileCtx, call *ast.CallExpr, std, helper string) {
	c.report(fc, call.Fun.Pos(), call.Fun.End(), CategoryStdlib,
		std+": use rkit."+helper, "Replace with rkit."+helper, fc.rkit(helper))
}

// reportStdlibCall replaces the whole call with rkit.helper(arg).
func (c *checker) reportStdlibCall(fc *fileCtx, call *ast.CallExpr, std, helper string, arg ast.Expr) {
	c.report(fc, call.Pos(), call.End(), CategoryStdlib,
		std+": use rkit."+helper, "Replace with rkit."+helper, fc.rkit(helper+"("+fc.text(arg)+")"), arg)
}

func (c *checker) isStringConst(e ast.Expr, want string) bool {
	tv, ok := c.pass.TypesInfo.Types[e]

	return ok && tv.Value != nil && tv.Value.Kind() == constant.String && constant.StringVal(tv.Value) == want
}

// intToStrHelper picks the rkit formatter for an (unnamed) integer type; named types would not
// convert implicitly.
func intToStrHelper(t types.Type) string {
	b, ok := t.(*types.Basic)
	if !ok {
		return ""
	}

	switch b.Kind() {
	case types.Int, types.UntypedInt:
		return "IntToStr"
	case types.Int32:
		return "Int32ToStr"
	case types.Int64:
		return "Int64ToStr"
	case types.Uint:
		return "UIntToStr"
	case types.Uint32:
		return "UInt32ToStr"
	case types.Uint64:
		return "UInt64ToStr"
	}

	return ""
}
