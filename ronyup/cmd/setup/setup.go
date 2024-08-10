package setup

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/spf13/cobra"
)

var opt = struct {
	DestinationDir string
	Force          bool
	ModulePath     string
	ProjectName    string
	Template       string
	Custom         map[string]string
}{}

func init() {
	flagSet := Cmd.Flags()
	flagSet.StringVarP(&opt.DestinationDir, "dst", "d", ".", "destination directory for the setup")
	flagSet.BoolVarP(&opt.Force, "force", "f", false, "clean destination directory before setup")
	flagSet.StringVarP(&opt.ModulePath, "module", "m", "github.com/your/repo", "module path")
	flagSet.StringVarP(&opt.ProjectName, "project", "p", "MyProject", "project name")
	flagSet.StringVarP(&opt.Template, "template", "t", "", "possible values: rony | kit")
	flagSet.StringToStringVarP(&opt.Custom, "custom", "c", map[string]string{}, "custom values for the template")
}

var Cmd = &cobra.Command{
	Use:                "setup",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := cmd.ParseFlags(args)
		if err != nil {
			return err
		}

		err = createDestination()
		if err != nil {
			return err
		}

		err = copyTemplate()
		if err != nil {
			return err
		}

		err = initGo()
		if err != nil {
			return err
		}

		return nil
	},
}

func createDestination() error {
	// get the absolute path to the output directory
	dstPath, err := filepath.Abs(opt.DestinationDir)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(dstPath, 0755) //nolint:gofumpt
	if !isEmptyDir(dstPath) {
		if !opt.Force {
			return fmt.Errorf("%s directory is not empty, use -f to force", dstPath)
		}

		_ = os.RemoveAll(dstPath)      //nolint:gofumpt
		_ = os.MkdirAll(dstPath, 0755) //nolint:gofumpt
	}

	return nil
}

func copyTemplate() error {
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

func initGo() error {
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
