package stubgengo

import (
	"strings"

	"github.com/clubpay/ronykit/desc"
)

type genImpl struct{}

func Generate(desc *desc.Stub) (string, error) {
	sb := &strings.Builder{}

	err := tplStub.Execute(sb, desc)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}
