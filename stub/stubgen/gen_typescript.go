package stubgen

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/stub/internal/tpl"
)

var _ GenEngine = (*typescriptGE)(nil)

func NewTypescriptEngine(cfg TypescriptConfig) GenEngine {
	return &typescriptGE{
		cfg: cfg,
	}
}

type TypescriptConfig struct {
	GenerateSWR bool
}

type typescriptGE struct {
	cfg TypescriptConfig
}

func (t typescriptGE) Generate(in *Input) ([]GeneratedFile, error) {
	stubSB := &strings.Builder{}
	swrHooksSB := &strings.Builder{}

	err := tpl.TSStub.Execute(stubSB, in)
	if err != nil {
		return nil, err
	}

	err = tpl.TSSWRHooks.Execute(swrHooksSB, in)
	if err != nil {
		return nil, err
	}

	files := []GeneratedFile{
		runPrettier(
			GeneratedFile{
				Filename: "stub.ts",
				Data:     utils.S2B(stubSB.String()),
			},
		),
	}

	if t.cfg.GenerateSWR {
		files = append(
			files,
			runPrettier(
				GeneratedFile{
					Filename: "swr.hooks.ts",
					Data:     utils.S2B(swrHooksSB.String()),
				},
			),
		)
	}

	return files, nil
}

func runPrettier(gf GeneratedFile) GeneratedFile {
	cmd := exec.Command(
		"npx", "prettier",
		"--stdin-filepath", gf.Filename,
	)
	cmd.Stdin = bytes.NewReader(gf.Data)
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cmd.Stdout = out
	cmd.Stderr = errBuf

	err := cmd.Run()
	if err != nil {
		fmt.Println(out.String())
		fmt.Println(errBuf.String())

		return gf
	}

	gf.Data = out.Bytes()

	return gf
}
