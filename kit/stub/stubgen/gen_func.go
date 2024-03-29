package stubgen

import (
	"strings"

	"github.com/clubpay/ronykit/kit/internal/tpl"
)

// GenFunc is the function which generates the final code. For example to generate
// golang code use GolangStub
type GenFunc func(in Input) (string, error)

func GolangStub(in Input) (string, error) {
	sb := &strings.Builder{}

	err := tpl.GoStub.Execute(sb, in)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

func TypeScriptStub(in Input) (string, error) {
	sb := &strings.Builder{}

	err := tpl.TsStub.Execute(sb, in)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}
