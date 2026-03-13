package tpl

import (
	_ "embed"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/clubpay/ronykit/kit/utils"
)

var (
	//go:embed go/stub.gotmpl
	goFileStub string
	GoStub     *template.Template

	//go:embed ts/stub.tstmpl
	tsFileStub string
	//go:embed ts/swr.hooks.tstmpl
	tsFileSWRHooks string
	TSStub         *template.Template
	TSSWRHooks     *template.Template
)

func init() {
	GoStub = template.Must(template.New("stub").Funcs(sprig.FuncMap()).Funcs(FuncMaps).Parse(goFileStub))
	TSStub = template.Must(template.New("stub").Funcs(sprig.FuncMap()).Funcs(FuncMaps).Parse(tsFileStub))
	TSSWRHooks = template.Must(template.New("swrHooks").Funcs(sprig.FuncMap()).Funcs(FuncMaps).Parse(tsFileSWRHooks))
}

var FuncMaps = map[string]any{
	// strQuote quotes each element in a string slice. Unlike sprig's quote (which wraps
	// individual strings), this operates on slices and uses Go's %q formatting.
	"strQuote": func(elems []string) []string {
		out := make([]string, len(elems))
		for i, e := range elems {
			out[i] = fmt.Sprintf("%q", e)
		}

		return out
	},

	// Case conversion using kit/utils (project-specific conventions).
	"lowerCamelCase":  utils.ToLowerCamel,
	"camelCase":       utils.ToCamel,
	"screamSnakeCase": utils.ToScreamingSnake,

	// Domain-specific type converters and helpers.
	"goType":              goType,
	"tsType":              tsType,
	"tsReplacePathParams": tsReplacePathParams,
}
