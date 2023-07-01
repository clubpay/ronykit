package stubgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/internal/tpl"
	"github.com/clubpay/ronykit/kit/utils"
)

func GolangStub(in Input) (string, error) {
	sb := &strings.Builder{}

	err := tpl.GoStub.Execute(sb, in)
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

type Input struct {
	desc.Stub
	Name string
	Pkg  string
}

// GenFunc is the function which generates the final code. For example to generate
// golang code use GolangStub
type GenFunc func(in Input) (string, error)

type Generator struct {
	cfg genConfig
}

func New(opt ...Option) *Generator {
	cfg := defaultConfig
	for _, o := range opt {
		o(&cfg)
	}

	return &Generator{
		cfg: cfg,
	}
}

func (g *Generator) Generate(descs ...desc.ServiceDesc) error {
	stubs := make([]*desc.Stub, 0, len(descs))
	for _, serviceDesc := range descs {
		stubDesc, err := serviceDesc.Desc().Stub(g.cfg.tags...)
		if err != nil {
			return err
		}

		stubs = append(stubs, stubDesc)
	}

	mergedStub := desc.MergeStubs(stubs...)
	rawContent, err := g.cfg.genFunc(
		Input{
			Stub: *mergedStub,
			Name: g.cfg.stubName,
			Pkg:  g.cfg.pkgName,
		},
	)
	if err != nil {
		return err
	}

	dirPath := filepath.Join(g.cfg.outputDir, g.cfg.folderName)
	_ = os.Mkdir(dirPath, os.ModePerm)

	return os.WriteFile(
		fmt.Sprintf("%s/stub.go", dirPath),
		utils.S2B(rawContent),
		os.ModePerm,
	)
}

func (g *Generator) MustGenerate(desc ...desc.ServiceDesc) {
	if err := g.Generate(desc...); err != nil {
		panic(err)
	}
}
