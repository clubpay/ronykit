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

func setupWorkspace(ctx context.Context, req WorkspaceRequest, log Logger) error {
	if req.Path == "" {
		return fmt.Errorf("workspace path is required")
	}

	req.Kind = rkit.Coalesce(req.Kind, KindBackend)

	switch req.Kind {
	case KindBackend, KindFullstack, KindFrontend:
	default:
		return fmt.Errorf(
			"invalid workspace kind %q: must be %q, %q or %q",
			req.Kind, KindBackend, KindFullstack, KindFrontend,
		)
	}

	req.Module = rkit.Coalesce(req.Module, "github.com/your/repo")
	if err := validateModulePath(req.Module); err != nil {
		return err
	}

	req.AppName = rkit.Coalesce(req.AppName, "myapp")

	resolved, err := resolveSkillSelection(req.Skills, req.Kind)
	if err != nil {
		return err
	}

	repoPath, err := filepath.Abs(req.Path)
	if err != nil {
		return err
	}

	if err := prepareWorkspaceDir(repoPath, req.Force); err != nil {
		return err
	}

	return copyWorkspaceTemplate(ctx, workspaceCopyInput{
		repoRoot:       repoPath,
		module:         req.Module,
		appName:        req.AppName,
		kind:           req.Kind,
		resolvedSkills: resolved,
	}, log)
}

func prepareWorkspaceDir(repoPath string, force bool) error {
	if force && filepath.Dir(repoPath) == repoPath {
		return fmt.Errorf("refusing to replace filesystem root %s", repoPath)
	}

	info, err := os.Stat(repoPath)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(repoPath, 0o755)
		}

		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s exists and is not a directory", repoPath)
	}

	if z.IsEmptyDir(repoPath) {
		return nil
	}

	if !force {
		return fmt.Errorf("%s directory is not empty, use -f/--force to replace", repoPath)
	}

	if err := os.RemoveAll(repoPath); err != nil {
		return err
	}

	return os.MkdirAll(repoPath, 0o755)
}

type workspaceCopyInput struct {
	repoRoot       string
	module         string
	appName        string
	kind           string
	resolvedSkills []string
}

func goRootRel(kind string) string {
	if kind == KindFullstack {
		return backendDir
	}

	return "."
}

func goModulePrefix(module, kind string) string {
	base := strings.TrimSuffix(module, "/")
	if kind == KindFullstack {
		return path.Join(base, backendDir)
	}

	return base
}

func workspaceCommonDestMapper(repoRoot, kind string) func(string) (string, bool) {
	return func(relPath string) (string, bool) {
		if skipHookForKind(relPath, kind) {
			return "", true
		}

		return filepath.Join(repoRoot, relPath), false
	}
}

// skipHookForKind reports whether relPath is a hook script that does not apply to kind
// (backend-* hooks on frontend-only workspaces, frontend-* hooks on backend-only workspaces).
func skipHookForKind(relPath, kind string) bool {
	rel := filepath.ToSlash(relPath)
	if path.Base(path.Dir(rel)) != "hooks" {
		return false
	}

	base := path.Base(rel)

	return (kind == KindFrontend && strings.HasPrefix(base, "backend-")) ||
		(kind == KindBackend && strings.HasPrefix(base, "frontend-"))
}

// BackendDestPrefix is where skeleton/backend/ is copied.
func BackendDestPrefix(repoRoot, kind string) string {
	return backendDestPrefix(repoRoot, kind)
}

func backendDestPrefix(repoRoot, kind string) string {
	if kind == KindFullstack {
		return filepath.Join(repoRoot, backendDir)
	}

	return repoRoot
}

func copyWorkspaceTemplate(ctx context.Context, in workspaceCopyInput, log Logger) error {
	templateInput := TemplateInput{
		ApplicationName: in.appName,
		RepositoryPath:  goModulePrefix(in.module, in.kind),
		RonyKitPath:     RonyKitModulePath,
		Kind:            in.kind,
		Skills:          selectedSkillInfos(in.resolvedSkills),
	}

	callback := func(filePath string, dir bool) {
		if dir {
			log.Println("DIR: ", filePath, "created")
		} else {
			log.Println("FILE: ", filePath, "created")
		}
	}

	if err := z.CopyDir(z.CopyDirParams{
		FS:             internal.Skeleton,
		SrcPathPrefix:  filepath.Join("skeleton", "workspace"),
		DestPathPrefix: in.repoRoot,
		TemplateInput:  templateInput,
		DestMapper:     workspaceCommonDestMapper(in.repoRoot, in.kind),
		Callback:       callback,
	}); err != nil {
		return fmt.Errorf("copy workspace skeleton: %w", err)
	}

	if hasBackend(in.kind) {
		dest := backendDestPrefix(in.repoRoot, in.kind)
		if err := z.CopyDir(z.CopyDirParams{
			FS:             internal.Skeleton,
			SrcPathPrefix:  filepath.Join("skeleton", "backend"),
			DestPathPrefix: dest,
			DestRoot:       in.repoRoot,
			TemplateInput:  templateInput,
			DestMapper: func(relPath string) (string, bool) {
				rel := filepath.ToSlash(relPath)
				if rel == "cmd/bundle" || strings.HasPrefix(rel, "cmd/bundle/") {
					return "", true
				}

				return filepath.Join(dest, relPath), false
			},
			Callback: callback,
		}); err != nil {
			return fmt.Errorf("copy backend skeleton: %w", err)
		}
	}

	if hasFrontend(in.kind) {
		if err := z.CopyDir(z.CopyDirParams{
			FS:             internal.Skeleton,
			SrcPathPrefix:  filepath.Join("skeleton", "frontend"),
			DestPathPrefix: filepath.Join(in.repoRoot, frontendDir),
			DestRoot:       in.repoRoot,
			TemplateInput:  templateInput,
			Callback:       callback,
		}); err != nil {
			return fmt.Errorf("copy frontend skeleton: %w", err)
		}
	}

	if err := installSkillLinks(in.repoRoot, "ronykit-framework", true, callback); err != nil {
		return err
	}

	if err := installSkills(log, in.repoRoot, in.resolvedSkills); err != nil {
		return err
	}

	log.Println("Workspace created successfully")

	if hasBackend(in.kind) {
		if err := initGoWorkspace(ctx, in); err != nil {
			return err
		}
	}

	p := z.RunCmdParams{Dir: in.repoRoot}

	isGitRepo, err := isGitRepository(in.repoRoot)
	if err == nil && !isGitRepo {
		_ = z.RunCmd(ctx, p, "git", "init")
		_ = z.RunCmd(ctx, p, "git", "add", ".")
		_ = z.RunCmd(ctx, p, "git", "commit", "-m", "Workspace created")
	}

	return nil
}

func initGoWorkspace(ctx context.Context, in workspaceCopyInput) error {
	goRoot := filepath.Join(in.repoRoot, goRootRel(in.kind))
	modulePrefix := goModulePrefix(in.module, in.kind)
	packages := []string{"pkg/i18n", "pkg/runner", "cmd/" + defaultBundleName}

	p := z.RunCmdParams{Dir: goRoot, ENV: map[string]string{"GOWORK": "off"}}
	if err := z.RunCmd(ctx, p, "go", "work", "init"); err != nil {
		return fmt.Errorf("go work init: %w", err)
	}

	for _, pkg := range packages {
		p = z.RunCmdParams{Dir: filepath.Join(goRoot, pkg)}
		if err := z.RunCmd(ctx, p, "go", "mod", "init", path.Join(modulePrefix, pkg)); err != nil {
			return fmt.Errorf("go mod init %s: %w", pkg, err)
		}

		if err := z.RunCmd(ctx, p, "go", "mod", "edit", "-go=1.25"); err != nil {
			return fmt.Errorf("go mod edit %s: %w", pkg, err)
		}

		if pkg == "cmd/"+defaultBundleName {
			if err := addLocalModuleReplacement(ctx, goRoot, p.Dir, modulePrefix, RunnerRelDir); err != nil {
				return err
			}
		}

		if err := z.RunCmd(ctx, p, "go", "mod", "tidy", "-e"); err != nil {
			return fmt.Errorf("go mod tidy %s: %w", pkg, err)
		}

		if err := z.RunCmd(ctx, p, "go", "work", "use", "."); err != nil {
			return fmt.Errorf("go work use %s: %w", pkg, err)
		}
	}

	return nil
}

func installSkills(log Logger, repoRoot string, skills []string) error {
	if len(skills) == 0 {
		log.Println("No optional agent skills installed")

		return nil
	}

	if err := copySkills(repoRoot, skills, func(filePath string, dir bool) {
		if !dir {
			log.Println("FILE: ", filePath, "created")
		}
	}); err != nil {
		return err
	}

	log.Printf(
		"Installed %d agent skill(s): %s\n",
		len(skills),
		strings.Join(skills, ", "),
	)

	return nil
}

func isGitRepository(dir string) (bool, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(filepath.Join(absPath, ".git"))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
