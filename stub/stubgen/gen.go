package stubgen

import (
	"fmt"
	"os"
	"strings"

	"github.com/clubpay/ronykit/desc"
	"github.com/clubpay/ronykit/internal/tpl"
	"github.com/clubpay/ronykit/utils"
)

func GolangStub(desc *desc.Stub) (string, error) {
	sb := &strings.Builder{}

	err := tpl.GoStub.Execute(sb, desc)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

func Generate(
	serviceDesc desc.ServiceDesc,
	genFunc func(stub *desc.Stub) (string, error),
	pkgName string, tags ...string,
) error {
	stubDesc, err := serviceDesc.Desc().Stub(
		strings.ToLower(pkgName), tags...,
	)
	if err != nil {
		return err
	}

	rawContent, err := genFunc(stubDesc)
	if err != nil {
		return err
	}

	return os.WriteFile(
		fmt.Sprintf("%s.go", strings.ToLower(pkgName)),
		utils.S2B(rawContent),
		os.ModePerm,
	)
}

func MustGenerate(
	serviceDesc desc.ServiceDesc,
	genFunc func(stub *desc.Stub) (string, error),
	pkgName string, tags ...string,
) {
	err := Generate(serviceDesc, genFunc, pkgName, tags...)
	if err != nil {
		panic(err)
	}
}
