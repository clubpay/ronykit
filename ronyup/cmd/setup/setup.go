package setup

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"ronyup/internal"

	"github.com/jessevdk/go-flags"
	"github.com/spf13/cobra"
)

type Options struct {
	DestinationDir string            `short:"d" long:"dst" default:"." description:"destination directory for the setup"`
	Force          bool              `short:"f" long:"force" description:"clean destination directory before setup"`
	ModulePath     string            `short:"m" long:"module" description:"module path"`
	ProjectName    string            `short:"p" long:"project" description:"project name"`
	Template       string            `short:"t" long:"template" default:"rony" description:"possible values: rony | kit"`
	Custom         map[string]string `short:"c" long:"custom" description:"custom values for the template"`
}

var Cmd = &cobra.Command{
	Use:                "setup",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		var opt Options
		_, err := flags.ParseArgs(&opt, os.Args)
		if err != nil {
			return err
		}

		err = createDestination(opt)
		if err != nil {
			return err
		}

		err = copyTemplate(opt)
		if err != nil {
			return err
		}

		err = initGo(opt)
		if err != nil {
			return err
		}

		return nil
	},
}

func createDestination(opt Options) error {
	// get the absolute path to the output directory
	dstPath, err := filepath.Abs(opt.DestinationDir)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(dstPath, 0755)
	if !isEmptyDir(dstPath) {
		if !opt.Force {
			return fmt.Errorf("%s directory is not empty, use -f to force", dstPath)
		}

		_ = os.RemoveAll(dstPath)
	}
	_ = os.MkdirAll(dstPath, 0755)

	return nil
}

func copyTemplate(opt Options) error {
	return fs.WalkDir(
		internal.Skeleton,
		fmt.Sprintf("skeleton/%s", opt.Template),
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if path == fmt.Sprintf("skeleton/%s", opt.Template) {
				return nil
			}

			fn := strings.TrimPrefix(path, fmt.Sprintf("skeleton/%s/", opt.Template))
			if d.IsDir() {
				fmt.Println("Creating directory", fn)

				return os.MkdirAll(filepath.Join(opt.DestinationDir, fn), 0755)
			}

			fmt.Println("Creating file", fn)

			in, err := fs.ReadFile(internal.Skeleton, path)
			if err != nil {
				return err
			}

			out, err := os.Create(filepath.Join(opt.DestinationDir, strings.TrimSuffix(fn, ".gotmpl")))
			if err != nil {
				return err
			}

			t, err := template.New("t1").Parse(string(in))
			if err != nil {
				return err
			}

			err = t.Execute(out, opt)
			if err != nil {
				return err
			}

			_ = out.Close()

			return nil
		},
	)
}

func initGo(opt Options) error {
	fmt.Println("Initializing go module", opt.ModulePath)
	cmd := exec.Command("go", "mod", "init", opt.ModulePath)
	cmd.Dir = opt.DestinationDir

	err := cmd.Run()
	if err != nil {
		return err
	}

	fmt.Println("Tidying go module ...")
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = opt.DestinationDir

	err = cmd.Run()
	if err != nil {
		return err
	}

	fmt.Println("Formatting go code ...")
	cmd = exec.Command("go", "fmt", "./...")
	cmd.Dir = opt.DestinationDir

	err = cmd.Run()
	if err != nil {
		return err
	}

	fmt.Println("GIT init ...")
	cmd = exec.Command("git", "init")
	cmd.Dir = opt.DestinationDir

	err = cmd.Run()
	if err != nil {
		return err
	}

	fmt.Println("Go generate ...")
	cmd = exec.Command("go", "generate", "./...")
	cmd.Dir = opt.DestinationDir

	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
