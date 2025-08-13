package text

import (
	"context"
	"fmt"

	"github.com/clubpay/ronykit/util"
	"github.com/spf13/cobra"
	"golang.org/x/text/language"
	"golang.org/x/text/message/pipeline"
	"golang.org/x/tools/go/packages"
)

var opt = struct {
	SourceLang     string
	Languages      []string
	DstDir         string
	Packages       []string
	GenPackageName string
}{}

func init() {
	flagSet := Cmd.Flags()
	flagSet.StringVarP(&opt.SourceLang, "src-lang", "s", "en-US", "source language")
	flagSet.StringSliceVarP(&opt.Languages, "dst-lang", "l", []string{"en-US", "fa-IR"}, "languages to generate")
	flagSet.StringVarP(&opt.DstDir, "out-dir", "o", ".", "output path")
	flagSet.StringSliceVarP(&opt.Packages, "packages", "p", []string{}, "packages to generate")
	flagSet.StringVarP(&opt.GenPackageName, "gen-package", "g", "./internal/i18n", "package name for generated files")
}

var wrap = func(err error, msg string) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %v", msg, err)
}

//nolint:gochecknoglobals
var Cmd = &cobra.Command{
	Use: "text",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if len(opt.Packages) == 0 {
			packages, err := getAllPackages()
			if err != nil {
				return wrap(err, "failed to get packages")
			}

			opt.Packages = packages
		}

		config := &pipeline.Config{
			SourceLanguage: language.English,
			Supported:      util.Map(opt.Languages, language.Make),
			Packages:       opt.Packages,
			Dir:            opt.DstDir,
			GenFile:        "catalog.go",
			GenPackage:     opt.GenPackageName,
		}

		state, err := pipeline.Extract(config)
		if err != nil {
			return wrap(err, "extract failed")
		}
		if err := state.Import(); err != nil {
			return wrap(err, "import failed")
		}
		if err := state.Merge(); err != nil {
			return wrap(err, "merge failed")
		}
		if err := state.Export(); err != nil {
			return wrap(err, "export failed")
		}

		return wrap(state.Generate(), "generation failed")
	},
}

func getAllPackages() ([]string, error) {
	cfg := &packages.Config{
		Context: context.Background(),
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedModule,
		Dir:     ".", // module root or any subdir
	}
	// ./... expands to all packages in the current module/workspace
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return nil, fmt.Errorf("packages.Load: %w", err)
	}
	var out []string
	seen := make(map[string]struct{})
	for _, p := range pkgs {
		if p.PkgPath == "" || len(p.GoFiles) == 0 {
			continue // skip synthetic or empty packages
		}
		if _, ok := seen[p.PkgPath]; ok {
			continue
		}
		seen[p.PkgPath] = struct{}{}
		out = append(out, p.PkgPath)
	}
	return out, nil
}
