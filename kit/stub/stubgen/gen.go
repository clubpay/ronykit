package stubgen

import (
	"fmt"
	"os"
	"strings"

	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/internal/tpl"
	"github.com/clubpay/ronykit/kit/utils"
)

func GolangStub(desc *desc.Stub) (string, error) {
	sb := &strings.Builder{}

	err := tpl.GoStub.Execute(sb, desc)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

// GenFunc is the function which generates the final code. For example to generate
// golang code use GolangStub
type GenFunc func(stub *desc.Stub) (string, error)

type Generator struct {
	desc    []desc.ServiceDesc
	genFunc GenFunc
	pkgName string
	tags    []string
}

func New(pkgName string, tags ...string) Generator {
	return Generator{
		pkgName: pkgName,
		tags:    tags,
	}
}

func (g *Generator) SetFunc(gf GenFunc) *Generator {
	g.genFunc = gf

	return g
}

func (g *Generator) Generate(desc ...desc.ServiceDesc) error {
	for _, serviceDesc := range desc {
		stubDesc, err := serviceDesc.Desc().Stub(
			strings.ToLower(g.pkgName), g.tags...,
		)
		if err != nil {
			return err
		}

		rawContent, err := g.genFunc(stubDesc)
		if err != nil {
			return err
		}

		_ = os.Mkdir(strings.ToLower(serviceDesc.Desc().Name), os.ModePerm)

		return os.WriteFile(
			fmt.Sprintf("./%s/stub.go", strings.ToLower(serviceDesc.Desc().Name)),
			utils.S2B(rawContent),
			os.ModePerm,
		)
	}

	return nil
}

func (g *Generator) MustGenerate(desc ...desc.ServiceDesc) {
	err := g.Generate(desc...)
	if err != nil {
		panic(err)
	}
}
