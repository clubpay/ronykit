package stubgen

import (
	"go/format"
	"strings"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/stub/internal/tpl"
)

// GenFunc is the function which generates the final code. For example to generate
// golang code use GolangStub
type GenFunc func(in *Input) (string, error)

func GolangStub(in *Input) (string, error) {
	sb := &strings.Builder{}

	err := tpl.GoStub.Execute(sb, in)
	if err != nil {
		return "", err
	}

	formattedContent, err := format.Source(utils.S2B(sb.String()))
	if err != nil {
		return "", err
	}

	return utils.B2S(formattedContent), nil
}

func TypeScriptStub(in *Input) (string, error) {
	sb := &strings.Builder{}

	err := tpl.TSStub.Execute(sb, in)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}
