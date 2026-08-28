package scaffold

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
)

func setupFeature(ctx context.Context, req FeatureRequest, log Logger) error {
	if req.StartDir == "" {
		return fmt.Errorf("workspace start directory is required")
	}

	if req.FeatureDir == "" {
		return fmt.Errorf("project directory is required")
	}

	if req.Template == "" {
		req.Template = "service"
	}

	if req.FeaturePrefix == "" {
		req.FeaturePrefix = DefaultFeaturePrefix
	}

	if req.FeatureName == "" {
		req.FeatureName = req.FeatureDir
	}

	goRoot, err := resolveGoWorkspace(req.StartDir)
	if err != nil {
		return err
	}

	module := req.Module
	if module == "" {
		module, err = detectGoModule(goRoot)
		if err != nil {
			return fmt.Errorf("could not auto-detect repository go module: %w", err)
		}
	}

	req.FeatureDir = strings.TrimPrefix(req.FeatureDir, "/")
	req.FeatureDir = strings.TrimPrefix(req.FeatureDir, req.FeaturePrefix)

	packagePath := FeaturePackagePath(req.FeaturePrefix, req.Template, req.FeatureDir, req.GroupByTemplate)

	log.Printf("Go workspace: %s\n", goRoot)
	log.Printf("Repository module: %s\n", module)

	if err := createFeatureDir(goRoot, packagePath, req.Force); err != nil {
		return err
	}

	if err := copyFeatureTemplate(ctx, goRoot, module, packagePath, req, log); err != nil {
		return err
	}

	if err := sideEffectImportModule(ctx, goRoot, module, packagePath, log); err != nil {
		return err
	}

	cmdCtx := WorkspaceContext{
		Log:        log,
		GoRoot:     goRoot,
		RepoModule: module,
	}
	if err := syncBundlesForFeature(cmdCtx, packagePath); err != nil {
		log.PrintErrf("Warning: could not sync bundles: %v\n", err)
	}

	return nil
}

func createFeatureDir(goRoot, packagePath string, force bool) error {
	projectPath := filepath.Join(goRoot, filepath.FromSlash(packagePath))

	_ = os.MkdirAll(projectPath, 0o755)
	if z.IsEmptyDir(projectPath) {
		return nil
	}

	if !force {
		return fmt.Errorf("%s directory is not empty, use -f to force", projectPath)
	}

	if err := os.RemoveAll(projectPath); err != nil {
		return err
	}

	return os.MkdirAll(projectPath, 0o755)
}

func copyFeatureTemplate(
	ctx context.Context,
	goRoot, module, packagePath string,
	req FeatureRequest,
	log Logger,
) error {
	destPath := filepath.Join(goRoot, filepath.FromSlash(packagePath))

	rkit.Assert(z.CopyDir(
		z.CopyDirParams{
			FS:             internal.Skeleton,
			SrcPathPrefix:  filepath.Join("skeleton", req.Template),
			DestPathPrefix: destPath,
			TemplateInput: TemplateInput{
				RepositoryPath: strings.TrimSuffix(module, "/"),
				PackagePath:    strings.Trim(packagePath, "/"),
				PackageName:    req.FeatureName,
				RonyKitPath:    RonyKitModulePath,
			},
			Callback: func(filePath string, dir bool) {
				if dir {
					log.Println("DIR: ", filePath, "created")
				} else {
					log.Println("FILE: ", filePath, "created")
				}
			},
		},
	))

	log.Println("Feature created successfully")
	log.Println("Feature path:", destPath)

	p := z.RunCmdParams{Dir: destPath}
	if err := z.RunCmd(ctx, p, "go", "mod", "init", path.Join(module, packagePath)); err != nil {
		return fmt.Errorf("go mod init: %w", err)
	}

	if err := z.RunCmd(ctx, p, "go", "mod", "edit", "-go=1.25"); err != nil {
		return fmt.Errorf("go mod edit: %w", err)
	}

	if err := z.RunCmd(ctx, p, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	if err := z.RunCmd(ctx, p, "go", "fmt", "./..."); err != nil {
		return fmt.Errorf("go fmt: %w", err)
	}

	work := z.RunCmdParams{Dir: goRoot}
	if err := z.RunCmd(ctx, work, "go", "work", "use", "./"+packagePath); err != nil {
		return fmt.Errorf("go work use: %w", err)
	}

	return nil
}

func sideEffectImportModule(
	ctx context.Context,
	goRoot, module, packagePath string,
	log Logger,
) error {
	featuresFilePath := filepath.Join(goRoot, "cmd", defaultBundleName, "features.go")

	content, err := os.ReadFile(featuresFilePath)
	if err != nil {
		log.PrintErrf("Warning: Could not read features.go: %v\n", err)

		return nil
	}

	importPath := fmt.Sprintf("\t_ \"%s/%s\"\n", module, packagePath)
	if strings.Contains(string(content), importPath) {
		log.Println("Import already exists in features.go")

		return nil
	}

	lines := strings.Split(string(content), "\n")

	var newContent strings.Builder

	importAdded := false

	for i, line := range lines {
		newContent.WriteString(line)

		if i < len(lines)-1 {
			newContent.WriteString("\n")
		}

		if !importAdded && strings.HasPrefix(strings.TrimSpace(line), "package main") {
			hasImport := false

			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "import") {
					hasImport = true

					break
				}

				if strings.TrimSpace(lines[j]) != "" &&
					!strings.HasPrefix(strings.TrimSpace(lines[j]), "//") {
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

		if !importAdded && strings.HasPrefix(strings.TrimSpace(line), "import (") {
			newContent.WriteString(importPath)

			importAdded = true
		}
	}

	if err := os.WriteFile(featuresFilePath, []byte(newContent.String()), 0o644); err != nil {
		log.PrintErrf("Warning: Could not write to features.go: %v\n", err)

		return nil
	}

	log.Println("Feature import added to features.go")

	p := z.RunCmdParams{Dir: filepath.Join(goRoot, "cmd", defaultBundleName)}
	if err := z.RunCmd(ctx, p, "go", "mod", "tidy"); err != nil {
		return fmt.Errorf("go mod tidy cmd/%s: %w", defaultBundleName, err)
	}

	if err := z.RunCmd(ctx, p, "go", "fmt", "./..."); err != nil {
		return fmt.Errorf("go fmt cmd/%s: %w", defaultBundleName, err)
	}

	return nil
}
