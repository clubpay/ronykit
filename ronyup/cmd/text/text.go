package text

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/clubpay/ronykit/util"
	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"
	"golang.org/x/text/language"
	"golang.org/x/text/message/pipeline"
	"golang.org/x/tools/go/packages"
)

var opt = struct {
	SourceLang     string
	Languages      []string
	DstDir         string
	Packages       []string
	ModulesFilter  []string
	GenFile        string
	GenPackageName string
}{}

func init() {
	flagSet := Cmd.Flags()
	flagSet.StringVarP(&opt.SourceLang, "src-lang", "s", "en-US", "source language")
	flagSet.StringSliceVarP(&opt.Languages, "dst-lang", "l", []string{"en-US", "fa-IR"}, "languages to generate")
	flagSet.StringVarP(&opt.DstDir, "out-dir", "o", ".", "output path")
	flagSet.StringSliceVarP(&opt.Packages, "packages", "p", []string{}, "packages to generate")
	flagSet.StringVarP(&opt.GenFile, "gen-file", "f", "catalog.go", "generated file name")
	flagSet.StringSliceVarP(&opt.ModulesFilter, "modules-filter", "m", []string{}, "modules filter")

	flagSet.StringVarP(&opt.GenPackageName, "gen-package", "g", "", "package name for generated files")
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
			packages, err := getAllPackages(cmd)
			if err != nil {
				return wrap(err, "failed to get packages")
			}

			opt.Packages = packages
		}

		config := &pipeline.Config{
			SourceLanguage: language.Make(opt.SourceLang),
			Supported:      util.Map(opt.Languages, language.Make),
			Packages:       opt.Packages,
			Dir:            opt.DstDir,
			GenFile:        opt.GenFile,
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

func isGoWorkspace() bool {
	_, err := os.Stat("go.work")
	return err == nil
}

func getAllWorkspaceDirectories(cmd *cobra.Command) ([]string, error) {
	workData, err := os.ReadFile("go.work")
	if err != nil {
		return nil, fmt.Errorf("read go.work: %w", err)
	}

	f, err := modfile.ParseWork("go.work", workData, nil)
	if err != nil {
		return nil, fmt.Errorf("parse go.work: %w", err)
	}

	dirs := make([]string, 0, len(f.Use))
	for _, u := range f.Use {
		if len(opt.ModulesFilter) > 0 {
			filtered := false
			for _, m := range opt.ModulesFilter {
				if u.Path == m {
					filtered = true

					break
				}
			}
			if !filtered {
				continue
			}
		}
		dirs = append(dirs, u.Path)
	}

	return dirs, nil
}

func getAllPackages(cmd *cobra.Command) ([]string, error) {
	var dirs []string
	if isGoWorkspace() {
		cmd.Println("detected go workspace")
		var err error
		dirs, err = getAllWorkspaceDirectories(cmd)
		if err != nil {
			return nil, err
		}

		cmd.Println("using directories:", dirs)
	} else {
		dirs = []string{"."}
	}

	var out []string
	for _, dir := range dirs {
		cfg := &packages.Config{
			Context: context.Background(),
			Mode:    packages.NeedName | packages.NeedFiles | packages.NeedModule,
			Dir:     dir, // module root or any subdir
		}
		// ./... expands to all packages in the current module/workspace
		pkgs, err := packages.Load(cfg, "./...")
		if err != nil {
			return nil, fmt.Errorf("packages.Load: %w", err)
		}

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
	}

	// Sort packages for consistent output
	sort.Strings(out)

	// Pretty print packages
	for _, pkg := range out {
		cmd.Printf("\t%s\n", pkg)
	}

	return out, nil
}
