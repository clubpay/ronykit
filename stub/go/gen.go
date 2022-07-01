package stubgengo

import (
	"strings"

	"github.com/clubpay/ronykit/desc"
)

type genImpl struct{}

func (g genImpl) Generate(desc desc.Stub) (string, error) {
	sb := &strings.Builder{}
	for _, dto := range desc.DTOs {
		err := tplDTO.Execute(sb, dto)
		if err != nil {
			return "", err
		}
	}

	return sb.String(), nil
}
