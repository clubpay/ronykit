package stubgen

import (
	"bytes"
	"context"
	"errors"
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
		return nil, fmt.Errorf("failed to execute stub template: %w", err)
	}

	err = tpl.TSSWRHooks.Execute(swrHooksSB, in)
	if err != nil {
		return nil, fmt.Errorf("failed to execute swr hooks template: %w", err)
	}

	stubFile, err := runPrettier(
		GeneratedFile{
			Filename: "stub.ts",
			Data:     utils.S2B(stubSB.String()),
		},
	)
	if err != nil {
		return nil, err
	}

	files := []GeneratedFile{stubFile}

	if t.cfg.GenerateSWR {
		swrFile, err := runPrettier(
			GeneratedFile{
				Filename: "swr.hooks.ts",
				Data:     utils.S2B(swrHooksSB.String()),
			},
		)
		if err != nil {
			return nil, err
		}

		files = append(files, swrFile)
	}

	return files, nil
}

func runPrettier(gf GeneratedFile) (GeneratedFile, error) {
	cmd := exec.CommandContext(
		context.Background(),
		"npx", "--yes", "prettier",
		"--stdin-filepath", gf.Filename,
	)
	cmd.Stdin = bytes.NewReader(gf.Data)
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	cmd.Stdout = out
	cmd.Stderr = errBuf

	err := cmd.Run()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return gf, nil
		}

		return gf, fmt.Errorf(`
prettier failed: %w

--- PRETTIER ERROR ---
%s
--- UNFORMATTED CODE ---
%s
------------------------
`, err, errBuf.String(), gf.Data)
	}

	gf.Data = out.Bytes()

	return gf, nil
}
