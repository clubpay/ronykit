package stubgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/stub/internal/tpl"
)

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
	in := NewInput(g.cfg.stubName, g.cfg.pkgName, descs...)
	in.AddTags(g.cfg.tags...)
	in.AddExtraOptions(g.cfg.extraOptions)

	rawContent, err := g.cfg.genFunc(in)
	if err != nil {
		return err
	}

	dirPath := filepath.Join(g.cfg.outputDir, g.cfg.folderName)
	_ = os.MkdirAll(dirPath, os.ModePerm) //nolint:errcheck

	return os.WriteFile(
		fmt.Sprintf("%s/stub.%s", dirPath, strings.TrimLeft(g.cfg.fileExtension, ".")),
		[]byte(rawContent),
		os.ModePerm,
	)
}

func (g *Generator) MustGenerate(desc ...desc.ServiceDesc) {
	if err := g.Generate(desc...); err != nil {
		panic(err)
	}
}

// AuxFunctions are a set of helper functions that could be used in
// your template
var AuxFunctions = tpl.FuncMaps
