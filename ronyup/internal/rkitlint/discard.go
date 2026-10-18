package rkitlint

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"regexp"

	"github.com/clubpay/ronykit/x/rkit"
)

// staleNolint matches a trailing comment that only silences dropped-error linters.
var staleNolint = regexp.MustCompile(`^//\s*nolint:(errcheck|errchkjson)(,(errcheck|errchkjson))*(\s.*)?$`)

// checkDiscard matches assignments that drop their second result:
//
//	v, _ := f()              → v := rkit.Ok(f())          (f returns (T, error))
//	n, _ := strconv.Atoi(s)  → n := rkit.StrToInt(s)      (and 64-bit ParseInt/ParseFloat)
//
// strconv.ParseUint stays on rkit.Ok: rkit.StrToUInt64 also accepts a leading '+'.
//
//	b, _ := json.Marshal(x)  → b := rkit.ToJSON(x)
//	v, _ := x.(T)            → v := rkit.TryCast[T](x)
func (c *checker) checkDiscard(fc *fileCtx, stmt ast.Stmt) {
	as, ok := stmt.(*ast.AssignStmt)
	if !ok || (as.Tok != token.DEFINE && as.Tok != token.ASSIGN) || len(as.Lhs) != 2 || len(as.Rhs) != 1 ||
		!isBlank(as.Lhs[1]) || isBlank(as.Lhs[0]) {
		return
	}

	assign := fc.text(as.Lhs[0]) + " = "
	if as.Tok == token.DEFINE {
		assign = fc.text(as.Lhs[0]) + " := "
	}

	// Nothing is dropped any more, so a trailing //nolint:errcheck would be reported as unused.
	end := as.End()
	if cg := fc.trailingComment(
		as,
	); cg != nil && len(cg.List) == 1 &&
		staleNolint.MatchString(cg.List[0].Text) {
		end = cg.End()
	}

	switch rhs := as.Rhs[0].(type) {
	case *ast.TypeAssertExpr:
		if rhs.Type == nil {
			return
		}

		c.report(fc, as.Pos(), end, CategoryDiscard,
			"type assertion with the ok result dropped: use rkit.TryCast",
			"Replace with rkit.TryCast",
			assign+fc.rkit("TryCast["+fc.text(rhs.Type)+"]("+fc.text(rhs.X)+")"), rhs)
	case *ast.CallExpr:
		res, isTuple := c.typeOf(rhs).(*types.Tuple)
		if !isTuple || res.Len() != 2 || !isError(res.At(1).Type()) {
			return
		}

		if helper, arg := c.errorFreeHelper(rhs); helper != "" {
			c.report(fc, as.Pos(), end, CategoryDiscard,
				"error dropped: use rkit."+helper,
				"Replace with rkit."+helper,
				assign+fc.rkit(helper+"("+fc.text(arg)+")"), arg)

			return
		}

		c.report(fc, as.Pos(), end, CategoryDiscard,
			"error dropped with _: use rkit.Ok",
			"Replace with rkit.Ok",
			assign+fc.rkit("Ok("+fc.text(rhs)+")"), rhs)
	}
}

// errorFreeHelper returns the rkit helper that makes call without the error, and its argument.
func (c *checker) errorFreeHelper(call *ast.CallExpr) (string, ast.Expr) {
	fn := c.callee(call)
	if fn == nil {
		return "", nil
	}

	args := call.Args

	switch fn.FullName() {
	case "strconv.Atoi":
		return "StrToInt", args[0]
	case "strconv.ParseInt":
		if c.isIntConst(args[1], 10) && c.isIntConst(args[2], 64) {
			return "StrToInt64", args[0]
		}
	case "strconv.ParseFloat":
		if c.isIntConst(args[1], 64) {
			return "StrToFloat64", args[0]
		}
	case "encoding/json.Marshal":
		return "ToJSON", args[0]
	}

	return "", nil
}

// callee returns the package-level function a call invokes as pkg.Func(...), or nil.
func (c *checker) callee(call *ast.CallExpr) *types.Func {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil
	}

	if id, isIdent := sel.X.(*ast.Ident); !isIdent {
		return nil
	} else if _, isPkg := c.pass.TypesInfo.Uses[id].(*types.PkgName); !isPkg {
		return nil
	}

	fn := rkit.TryCast[*types.Func](c.pass.TypesInfo.Uses[sel.Sel])

	return fn
}

func (c *checker) isIntConst(e ast.Expr, want int64) bool {
	tv, ok := c.pass.TypesInfo.Types[e]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.Int {
		return false
	}

	v, exact := constant.Int64Val(tv.Value)

	return exact && v == want
}
