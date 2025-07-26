package setup

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/ronyup/internal/z"
	"github.com/spf13/cobra"
)

var opt = struct {
	RepositoryRootDir  string
	RepositoryGoModule string
	ProjectDir         string
	ProjectName        string

	Force    bool
	Template string
	Custom   map[string]string
}{}

func init() {
	flagSet := Cmd.Flags()
	flagSet.StringVarP(&opt.RepositoryRootDir, "repoDir", "d", "./my-repo", "destination directory for the setup")
	flagSet.StringVarP(&opt.RepositoryGoModule, "repoModule", "r", "github.com/your/repo", "go module for the repository")
	flagSet.StringVarP(&opt.ProjectDir, "projectDir", "p", "my-project", "destination directory inside repoDir for the setup")
	flagSet.StringVarP(&opt.ProjectName, "projectName", "n", "MyProject", "project name")

	flagSet.BoolVarP(&opt.Force, "force", "f", false, "clean destination directory before setup")
	flagSet.StringVarP(&opt.Template, "template", "t", "service", "possible values: service | gateway")
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

		copyTemplate(cmd)

		return nil
	},
}

func createDestination() error {
	// get the absolute path to the output directory
	repoPath, err := filepath.Abs(opt.RepositoryRootDir)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(repoPath, 0755) //nolint:gofumpt

	if opt.ProjectDir != "" {
		p := z.RunCmdParams{Dir: repoPath}
		z.RunCmd(p, "go", "work", "init")

		projectPath := filepath.Join(repoPath, opt.ProjectDir)
		_ = os.MkdirAll(projectPath, 0755) //nolint:gofumpt
		if !z.IsEmptyDir(projectPath) {
			if !opt.Force {
				return fmt.Errorf("%s directory is not empty, use -f to force", projectPath)
			}

			_ = os.RemoveAll(projectPath)      //nolint:gofumpt
			_ = os.MkdirAll(projectPath, 0755) //nolint:gofumpt
		}
	}

	return nil
}

type ModuleInput struct {
	RepositoryPath string
	// PackagePath is the folder that module will reside inside the Repository root folder
	PackagePath string
	// PackageName is the name of the package to be used for some internal variables
	PackageName string
	// RonyKitPath is the address of the RonyKIT modules
	RonyKitPath string
}

func copyTemplate(cmd *cobra.Command) {
	pathPrefix := filepath.Join("skeleton", opt.Template)
	packagePath := filepath.Join(opt.RepositoryRootDir, opt.ProjectDir)

	err := fs.WalkDir(
		internal.Skeleton, pathPrefix,
		func(currPath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			srcPath := strings.TrimPrefix(currPath, pathPrefix)
			destPath := strings.TrimSuffix(
				filepath.Join(".", packagePath, srcPath),
				"tmpl",
			)

			if d.IsDir() {
				// Create a directory if it doesn't exist
				return os.MkdirAll(destPath, os.ModePerm)
			}

			cmd.Println("FILE: ", destPath, "created")

			return z.Copy(z.CopyParams{
				FS:             internal.Skeleton,
				SrcPath:        currPath,
				DestPath:       destPath,
				TemplateSuffix: "tmpl",
				TemplateInput: ModuleInput{
					RepositoryPath: strings.TrimSuffix(opt.RepositoryRootDir, "/"),
					PackagePath:    strings.Trim(packagePath, "/"),
					PackageName:    opt.ProjectName,
				},
			})
		})
	if err != nil {
		panic(err)
	}

	cmd.Println("Module created successfully")
	cmd.Println("Project path: ", packagePath)
	p := z.RunCmdParams{Dir: filepath.Join(".", packagePath)}
	z.RunCmd(p, "go", "mod", "init", path.Join(opt.RepositoryRootDir, packagePath))
	z.RunCmd(p, "go", "mod", "edit", "-go=1.23")
	z.RunCmd(p, "go", "mod", "tidy")
	z.RunCmd(p, "go", "fmt", "./...")
	z.RunCmd(p, "go", "work", "use", ".")
}
