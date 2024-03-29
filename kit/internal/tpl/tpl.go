package tpl

import (
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/clubpay/ronykit/kit/utils"
)

var (
	//go:embed go/stub.gotmpl
	goFileStub string
	GoStub     *template.Template

	//go:embed ts/stub.tstmpl
	tsFileStub string
	TsStub     *template.Template
)

func init() {
	GoStub = template.Must(template.New("stub").Funcs(funcMaps).Parse(goFileStub))
	TsStub = template.Must(template.New("stub").Funcs(funcMaps).Parse(tsFileStub))
}

var funcMaps = map[string]any{
	"strQuote": func(elems []string) []string {
		out := make([]string, len(elems))
		for i, e := range elems {
			out[i] = fmt.Sprintf("%q", e)
		}

		return out
	},
	"strJoin":         strings.Join,
	"strSplit":        strings.Split,
	"toUpper":         strings.ToUpper,
	"toLower":         strings.ToLower,
	"toTitle":         strings.ToTitle,
	"toUTF8":          strings.ToValidUTF8,
	"lowerCamelCase":  utils.ToLowerCamel,
	"camelCase":       utils.ToCamel,
	"kebabCase":       utils.ToKebab,
	"snakeCase":       utils.ToSnake,
	"screamSnakeCase": utils.ToScreamingSnake,
	"randomID":        utils.RandomID,
	"randomDigit":     utils.RandomDigit,
	"randomInt":       utils.RandomInt64,
	"goType":          GoType,
}
