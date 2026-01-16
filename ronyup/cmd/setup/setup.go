package setup

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/ronyup/internal/z"
	"github.com/clubpay/ronykit/x/rkit"
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
	rootFlagSet := Cmd.PersistentFlags()
	rootFlagSet.StringVarP(
		&opt.RepositoryRootDir,
		"repoDir",
		"d",
		"./my-repo",
		"destination directory for the setup",
	)
	rootFlagSet.StringVarP(
		&opt.RepositoryGoModule,
		"repoModule",
		"r",
		"github.com/your/repo",
		"go module for the repository",
	)
	rootFlagSet.BoolVarP(&opt.Force, "force", "f", false, "clean destination directory before setup")
	rootFlagSet.StringToStringVarP(&opt.Custom, "custom", "c", map[string]string{}, "custom values for the template")

	projectFlagSet := CmdSetupProject.Flags()
	projectFlagSet.StringVarP(
		&opt.ProjectDir,
		"projectDir",
		"p",
		"my-project",
		"destination directory inside repoDir for the setup",
	)
	projectFlagSet.StringVarP(&opt.ProjectName, "projectName", "n", "MyProject", "project name")
	projectFlagSet.StringVarP(
		&opt.Template,
		"template",
		"t",
		"service",
		"possible values: service | job",
	)

	Cmd.AddCommand(CmdSetupWorkspace, CmdSetupProject)
}

var Cmd = &cobra.Command{
	Use:                "setup",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := cmd.ParseFlags(args)
		if err != nil {
			return err
		}

		return nil
	},
}

var CmdSetupWorkspace = &cobra.Command{
	Use:                "workspace",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := cmd.ParseFlags(args)
		if err != nil {
			return err
		}

		err = createWorkspace(cmd.Context())
		if err != nil {
			return err
		}

		copyWorkspaceTemplate(cmd)

		return nil
	},
}

var CmdSetupProject = &cobra.Command{
	Use:                "service",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := cmd.ParseFlags(args)
		if err != nil {
			return err
		}

		err = createProject(cmd.Context())
		if err != nil {
			return err
		}

		copyProjectTemplate(cmd)

		return nil
	},
}

func createWorkspace(_ context.Context) error {
	// get the absolute path to the output directory
	repoPath, err := filepath.Abs(opt.RepositoryRootDir)
	if err != nil {
		return err
	}

	_ = os.MkdirAll(repoPath, 0o755)

	return nil
}

func createProject(_ context.Context) error {
	if opt.ProjectDir == "" {
		return fmt.Errorf("project directory is required")
	}

	projectPath := filepath.Join("feature", opt.ProjectDir)

	_ = os.MkdirAll(projectPath, 0o755)
	if !z.IsEmptyDir(projectPath) {
		if !opt.Force {
			return fmt.Errorf("%s directory is not empty, use -f to force", projectPath)
		}

		rkit.Assert(os.RemoveAll(projectPath))
		rkit.Assert(os.MkdirAll(projectPath, 0o755))
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

func copyProjectTemplate(cmd *cobra.Command) {
	pathPrefix := filepath.Join("skeleton", opt.Template)
	packagePath := filepath.Join("feature", opt.ProjectDir)

	rkit.Assert(
		fs.WalkDir(
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
						RepositoryPath: strings.TrimSuffix(opt.RepositoryGoModule, "/"),
						PackagePath:    strings.Trim(path.Join("feature", opt.ProjectDir), "/"),
						PackageName:    opt.ProjectName,
						RonyKitPath:    "github.com/clubpay/ronykit",
					},
				})
			}),
	)

	cmd.Println("Project created successfully")
	cmd.Println("Project path:", packagePath)
	p := z.RunCmdParams{Dir: filepath.Join("./feature", packagePath)}
	z.RunCmd(cmd.Context(), p, "go", "mod", "init", path.Join(opt.RepositoryGoModule, "feature", opt.ProjectDir))
	z.RunCmd(cmd.Context(), p, "go", "mod", "edit", "-go=1.25")
	z.RunCmd(cmd.Context(), p, "go", "mod", "tidy")
	z.RunCmd(cmd.Context(), p, "go", "fmt", "./...")
	z.RunCmd(cmd.Context(), p, "go", "work", "use", ".")
}

func copyWorkspaceTemplate(cmd *cobra.Command) {
	pathPrefix := filepath.Join("skeleton", "workspace")

	rkit.Assert(
		fs.WalkDir(
			internal.Skeleton, pathPrefix,
			func(currPath string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				srcPath := strings.TrimPrefix(currPath, pathPrefix)
				destPath := strings.TrimSuffix(
					filepath.Join(".", opt.RepositoryRootDir, srcPath),
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
						RepositoryPath: strings.TrimSuffix(opt.RepositoryGoModule, "/"),
						PackagePath:    strings.Trim(opt.ProjectDir, "/"),
						PackageName:    opt.ProjectName,
						RonyKitPath:    "github.com/clubpay/ronykit",
					},
				})
			}),
	)

	cmd.Println("Workspace created successfully")
	p := z.RunCmdParams{Dir: filepath.Join(".", opt.RepositoryRootDir)}
	z.RunCmd(cmd.Context(), p, "go", "work", "init")
	p = z.RunCmdParams{Dir: filepath.Join(".", opt.RepositoryRootDir, "pkg/i18n")}
	z.RunCmd(cmd.Context(), p, "go", "mod", "init", path.Join(opt.RepositoryGoModule, "pkg/i18n"))
	z.RunCmd(cmd.Context(), p, "go", "mod", "edit", "-go=1.25")
	z.RunCmd(cmd.Context(), p, "go", "mod", "tidy")
	z.RunCmd(cmd.Context(), p, "go", "work", "use", ".")
}
