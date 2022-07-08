package stubgengo

import (
	_ "embed"
	"strings"
	"text/template"

	"github.com/clubpay/ronykit/utils"
)

var (
	//go:embed tpl/stub.gotmpl
	tplFileStub string
	tplStub     *template.Template
)

func init() {
	tplStub = template.Must(template.New("stub").Funcs(funcMaps).Parse(tplFileStub))
}

var funcMaps = map[string]interface{}{
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
}
