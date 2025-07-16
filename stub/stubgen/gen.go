package stubgen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/stub/internal/tpl"
)

type GeneratedFile struct {
	SubFolder string
	Filename  string
	Data      []byte
}

type GenEngine interface {
	Generate(in *Input) ([]GeneratedFile, error)
}

type GenFunc func(in *Input) ([]GeneratedFile, error)

func (f GenFunc) Generate(in *Input) ([]GeneratedFile, error) {
	return f(in)
}

var _ GenEngine = GenFunc(nil)

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
	in := NewInput(g.cfg.stubName, descs...)
	in.AddTags(g.cfg.tags...)

	files, err := g.cfg.genEngine.Generate(in)
	if err != nil {
		return err
	}

	dirPath := filepath.Join(g.cfg.outputDir, g.cfg.folderName)
	_ = os.MkdirAll(dirPath, os.ModePerm) //nolint:errcheck

	for _, file := range files {
		err = os.WriteFile(
			fmt.Sprintf("%s/%s", dirPath, file.Filename),
			file.Data,
			os.ModePerm,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) MustGenerate(desc ...desc.ServiceDesc) {
	err := g.Generate(desc...)
	if err != nil {
		panic(err)
	}
}

// AuxFunctions are a set of helper functions that could be used in
// your template
var AuxFunctions = tpl.FuncMaps
