package setup

import (
	"context"
	"fmt"
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
	ApplicationName    string
	RepositoryRootDir  string
	RepositoryGoModule string
	FeatureDir         string
	FeatureName        string

	Force    bool
	Template string
	Custom   map[string]string
}{}

func init() {
	rootFlagSet := Cmd.PersistentFlags()
	rootFlagSet.StringVarP(
		&opt.RepositoryGoModule,
		"repoModule",
		"m",
		"github.com/your/repo",
		"go module for the repository",
	)
	rootFlagSet.BoolVarP(&opt.Force, "force", "f", false, "clean destination directory before setup")
	rootFlagSet.StringToStringVarP(&opt.Custom, "custom", "c", map[string]string{}, "custom values for the template")

	featureFlagSet := CmdSetupFeature.Flags()
	featureFlagSet.StringVarP(
		&opt.FeatureDir,
		"featureDir",
		"p",
		"my_feature",
		"destination directory inside repoDir for the setup",
	)
	featureFlagSet.StringVarP(
		&opt.FeatureName,
		"featureName",
		"n",
		"myfeature",
		"feature name",
	)
	featureFlagSet.StringVarP(
		&opt.Template,
		"template",
		"t",
		"service",
		"possible values: service | job | gateway",
	)

	_ = CmdSetupFeature.RegisterFlagCompletionFunc(
		"template",
		func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return []string{"service", "job", "gateway"}, cobra.ShellCompDirectiveNoFileComp
		},
	)

	workspaceFlagSet := CmdSetupWorkspace.Flags()
	workspaceFlagSet.StringVarP(
		&opt.RepositoryRootDir,
		"repoDir",
		"r",
		"./my-repo",
		"destination directory for the setup",
	)
	workspaceFlagSet.StringVarP(
		&opt.ApplicationName,
		"appName",
		"a",
		"myapp",
		"application name",
	)

	Cmd.AddCommand(CmdSetupWorkspace, CmdSetupFeature)
}

var Cmd = &cobra.Command{
	Use:                "setup",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 && cmd.Flags().NFlag() == 0 {
			err := RunInteractive(cmd)
			if err != nil {
				return err
			}
		}

		return nil
	},
}

type TemplateInput struct {
	ApplicationName string
	RepositoryPath  string
	// PackagePath is the folder that module will reside inside the Repository root folder
	PackagePath string
	// PackageName is the name of the package to be used for some internal variables
	PackageName string
	// RonyKitPath is the address of the RonyKIT modules
	RonyKitPath string
}

var CmdSetupWorkspace = &cobra.Command{
	Use:                "workspace",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := cmd.ParseFlags(args)
		if err != nil {
			return err
		}

		return runWorkspace(cmd)
	},
}

func runWorkspace(cmd *cobra.Command) error {
	if err := createWorkspace(cmd.Context()); err != nil {
		return err
	}

	copyWorkspaceTemplate(cmd)

	return nil
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

func copyWorkspaceTemplate(cmd *cobra.Command) {
	pathPrefix := filepath.Join("skeleton", "workspace")

	rkit.Assert(z.CopyDir(
		z.CopyDirParams{
			FS:             internal.Skeleton,
			SrcPathPrefix:  pathPrefix,
			DestPathPrefix: filepath.Join(".", opt.RepositoryRootDir),
			TemplateInput: TemplateInput{
				ApplicationName: opt.ApplicationName,
				RepositoryPath:  strings.TrimSuffix(opt.RepositoryGoModule, "/"),
				PackagePath:     strings.Trim(opt.FeatureDir, "/"),
				PackageName:     opt.FeatureName,
				RonyKitPath:     "github.com/clubpay/ronykit",
			},
			Callback: func(filePath string, dir bool) {
				if dir {
					cmd.Println("DIR: ", filePath, "created")
				} else {
					cmd.Println("FILE: ", filePath, "created")
				}
			},
		},
	))

	cmd.Println("Workspace created successfully")

	packages := []string{"pkg/i18n", "cmd/service"}
	p := z.RunCmdParams{Dir: filepath.Join(".", opt.RepositoryRootDir)}
	z.RunCmd(cmd.Context(), p, "go", "work", "init")

	for _, pkg := range packages {
		p = z.RunCmdParams{Dir: filepath.Join(".", opt.RepositoryRootDir, pkg)}
		z.RunCmd(cmd.Context(), p, "go", "mod", "init", path.Join(opt.RepositoryGoModule, pkg))
		z.RunCmd(cmd.Context(), p, "go", "mod", "edit", "-go=1.25")
		z.RunCmd(cmd.Context(), p, "go", "mod", "tidy", "-e")
		z.RunCmd(cmd.Context(), p, "go", "work", "use", ".")
	}

	p = z.RunCmdParams{Dir: filepath.Join(".", opt.RepositoryRootDir)}

	isGitRepo, err := isGitRepository(filepath.Join(".", opt.RepositoryRootDir))
	if err == nil && !isGitRepo {
		z.RunCmd(cmd.Context(), p, "git", "init")
		z.RunCmd(cmd.Context(), p, "git", "add", ".")
		z.RunCmd(cmd.Context(), p, "git", "commit", "-m", "Workspace created")
	}
}

func isGitRepository(dir string) (bool, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}

	gitPath := filepath.Join(absPath, ".git")

	_, err = os.Stat(gitPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

var CmdSetupFeature = &cobra.Command{
	Use:                "feature",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		err := cmd.ParseFlags(args)
		if err != nil {
			return err
		}

		return runFeature(cmd)
	},
}

func runFeature(cmd *cobra.Command) error {
	cwd := rkit.GetCurrentDir()

	ok, err := isGoWorkspaceRoot(cwd)
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("run this command in a go workspace root directory")
	}

	opt.RepositoryRootDir = cwd

	if f := cmd.Flag("repoModule"); f == nil || !f.Changed {
		detected, err := detectGoModule(cwd)
		if err != nil {
			return fmt.Errorf("could not auto-detect repository go module: %w", err)
		}

		opt.RepositoryGoModule = detected
	}

	cmd.Printf("Repository module: %s\n", opt.RepositoryGoModule)

	if err := createFeature(cmd.Context()); err != nil {
		return err
	}

	copyFeatureTemplate(cmd)
	sideEffectImportModule(cmd)

	return nil
}

func isGoWorkspaceRoot(dir string) (bool, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}

	goWorkPath := filepath.Join(absPath, "go.work")

	_, err = os.Stat(goWorkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// detectGoModule reads go.work to find use directives, then reads the first
// resolvable go.mod to derive the repository root module path.
func detectGoModule(workspaceDir string) (string, error) {
	goWorkPath := filepath.Join(workspaceDir, "go.work")

	data, err := os.ReadFile(goWorkPath)
	if err != nil {
		return "", fmt.Errorf("could not read go.work: %w", err)
	}

	useDirs := parseUseDirectives(string(data))
	if len(useDirs) == 0 {
		return "", fmt.Errorf("no use directive found in go.work")
	}

	for _, useDir := range useDirs {
		goModPath := filepath.Join(workspaceDir, filepath.FromSlash(useDir), "go.mod")

		modData, err := os.ReadFile(goModPath)
		if err != nil {
			continue
		}

		modulePath := parseModulePath(string(modData))
		if modulePath == "" {
			continue
		}

		relDir := path.Clean(strings.TrimPrefix(filepath.ToSlash(useDir), "./"))
		if relDir == "." {
			return modulePath, nil
		}

		rootModule := strings.TrimSuffix(modulePath, "/"+relDir)
		if rootModule != modulePath {
			return rootModule, nil
		}
	}

	return "", fmt.Errorf("could not derive repository module from go.work entries")
}

func parseUseDirectives(goWorkContent string) []string {
	var dirs []string

	lines := strings.Split(goWorkContent, "\n")
	inUseBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inUseBlock {
			if trimmed == ")" {
				inUseBlock = false

				continue
			}

			if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
				dirs = append(dirs, trimmed)
			}

			continue
		}

		if trimmed == "use (" {
			inUseBlock = true

			continue
		}

		if strings.HasPrefix(trimmed, "use ") && !strings.Contains(trimmed, "(") {
			dir := strings.TrimSpace(strings.TrimPrefix(trimmed, "use "))
			if dir != "" {
				dirs = append(dirs, dir)
			}
		}
	}

	return dirs
}

func parseModulePath(goModContent string) string {
	for line := range strings.SplitSeq(goModContent, "\n") {
		trimmed := strings.TrimSpace(line)
		if mod, ok := strings.CutPrefix(trimmed, "module "); ok {
			return strings.TrimSpace(mod)
		}
	}

	return ""
}

func createFeature(_ context.Context) error {
	if opt.FeatureDir == "" {
		return fmt.Errorf("project directory is required")
	}

	opt.FeatureDir = strings.TrimPrefix(opt.FeatureDir, "/")
	opt.FeatureDir = strings.TrimPrefix(opt.FeatureDir, "feature")
	projectPath := filepath.Join("feature", opt.Template, opt.FeatureDir)

	_ = os.MkdirAll(projectPath, 0o755)
	if z.IsEmptyDir(projectPath) {
		return nil
	}

	if !opt.Force {
		return fmt.Errorf("%s directory is not empty, use -f to force", projectPath)
	}

	rkit.Assert(os.RemoveAll(projectPath))
	rkit.Assert(os.MkdirAll(projectPath, 0o755))

	return nil
}

func copyFeatureTemplate(cmd *cobra.Command) {
	pathPrefix := filepath.Join("skeleton/feature", opt.Template)
	packagePath := filepath.Join("feature", opt.Template, opt.FeatureDir)

	rkit.Assert(z.CopyDir(
		z.CopyDirParams{
			FS:             internal.Skeleton,
			SrcPathPrefix:  pathPrefix,
			DestPathPrefix: filepath.Join(".", packagePath),
			TemplateInput: TemplateInput{
				RepositoryPath: strings.TrimSuffix(opt.RepositoryGoModule, "/"),
				PackagePath:    strings.Trim(path.Join(packagePath), "/"),
				PackageName:    opt.FeatureName,
				RonyKitPath:    "github.com/clubpay/ronykit",
			},
			Callback: func(filePath string, dir bool) {
				if dir {
					cmd.Println("DIR: ", filePath, "created")
				} else {
					cmd.Println("FILE: ", filePath, "created")
				}
			},
		},
	))

	cmd.Println("Feature created successfully")
	cmd.Println("Feature path:", packagePath)
	p := z.RunCmdParams{Dir: filepath.Join(packagePath)}
	z.RunCmd(cmd.Context(), p, "go", "mod", "init", path.Join(opt.RepositoryGoModule, packagePath))
	z.RunCmd(cmd.Context(), p, "go", "mod", "edit", "-go=1.25")
	z.RunCmd(cmd.Context(), p, "go", "mod", "tidy")
	z.RunCmd(cmd.Context(), p, "go", "fmt", "./...")
	z.RunCmd(cmd.Context(), p, "go", "work", "use", ".")
}

func sideEffectImportModule(cmd *cobra.Command) {
	featuresFilePath := filepath.Join(".", "cmd", "service", "features.go")

	// Read the existing file
	content, err := os.ReadFile(featuresFilePath)
	if err != nil {
		cmd.PrintErrf("Warning: Could not read features.go: %v\n", err)

		return
	}

	// Create the import statement for the feature
	packagePath := filepath.Join("feature", opt.Template, opt.FeatureDir)
	importPath := fmt.Sprintf("\t_ \"%s/%s\"\n", opt.RepositoryGoModule, packagePath)

	// Check if the import already exists
	if strings.Contains(string(content), importPath) {
		cmd.Println("Import already exists in features.go")

		return
	}

	lines := strings.Split(string(content), "\n")

	var newContent strings.Builder

	importAdded := false

	for i, line := range lines {
		newContent.WriteString(line)

		if i < len(lines)-1 {
			newContent.WriteString("\n")
		}

		// Add import after "package main" declaration
		if !importAdded && strings.HasPrefix(strings.TrimSpace(line), "package main") {
			// Check if the import block exists
			hasImport := false

			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "import") {
					hasImport = true

					break
				}

				if strings.TrimSpace(lines[j]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[j]), "//") {
					break
				}
			}

			if !hasImport {
				newContent.WriteString("\n")
				newContent.WriteString("import (\n")
				newContent.WriteString(importPath)
				newContent.WriteString(")\n")

				importAdded = true
			}
		}

		// Add to the existing import block
		if !importAdded && strings.HasPrefix(strings.TrimSpace(line), "import (") {
			newContent.WriteString(importPath)

			importAdded = true
		}
	}

	// Write back to the file
	err = os.WriteFile(featuresFilePath, []byte(newContent.String()), 0o644)
	if err != nil {
		cmd.PrintErrf("Warning: Could not write to features.go: %v\n", err)

		return
	}

	cmd.Println("Feature import added to features.go")

	p := z.RunCmdParams{Dir: filepath.Join("./cmd/service")}
	z.RunCmd(cmd.Context(), p, "go", "mod", "tidy")
	z.RunCmd(cmd.Context(), p, "go", "fmt", "./...")
}
