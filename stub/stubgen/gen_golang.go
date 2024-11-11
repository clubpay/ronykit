package stubgen

import (
	"go/format"
	"strings"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/stub/internal/tpl"
)

var _ GenEngine = (*golangGE)(nil)

func NewGolangEngine(cfg GolangConfig) GenEngine {
	ge := &golangGE{
		cfg: cfg,
	}

	if ge.cfg.Filename == "" {
		ge.cfg.Filename = "stub.go"
	}

	return ge
}

type GolangConfig struct {
	PkgName  string
	Filename string
}

type golangGE struct {
	cfg GolangConfig
}

func (g golangGE) Generate(in *Input) ([]GeneratedFile, error) {
	in.pkg = g.cfg.PkgName

	sb := &strings.Builder{}

	err := tpl.GoStub.Execute(sb, in)
	if err != nil {
		return nil, err
	}

	formattedContent, err := format.Source(utils.S2B(sb.String()))
	if err != nil {
		return nil, err
	}

	return []GeneratedFile{
		{
			SubFolder: "",
			Filename:  g.cfg.Filename,
			Data:      formattedContent,
		},
	}, nil
}
