package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/ronyup/internal/z"
	"github.com/spf13/cobra"
)

// Sync section identifiers for --only.
const (
	syncSectionAgents   = "agents"
	syncSectionAI       = "ai"
	syncSectionHooks    = "hooks"
	syncSectionDevops   = "devops"
	syncSectionDocs     = "docs"
	syncSectionSkills   = "skills"
	syncSectionBackend  = "backend"
	syncSectionFrontend = "frontend"
	syncSectionAll      = "all"
	syncKindAuto        = "auto"
	skillSyncInstalled  = "installed"
)

var syncOpt = struct {
	RepoDir    string
	Kind       string
	Only       []string
	Overwrite  bool
	SkillsMode []string
}{}

func init() {
	flags := CmdSetupSync.Flags()
	flags.StringVarP(&syncOpt.RepoDir, "repoDir", "r", ".", "repository root to sync (default: current directory)")
	flags.StringVarP(&syncOpt.Kind, "kind", "k", syncKindAuto, "workspace kind: auto | backend | fullstack | frontend")
	flags.StringSliceVarP(
		&syncOpt.Only,
		"only",
		"o",
		[]string{syncSectionAll},
		"sections to refresh: all | agents | ai | hooks | devops | docs | skills | backend | frontend",
	)
	flags.BoolVar(
		&syncOpt.Overwrite,
		"overwrite",
		false,
		"replace existing scaffold files (default: add missing files only)",
	)
	flags.StringSliceVarP(
		&syncOpt.SkillsMode,
		"skills",
		"s",
		[]string{skillSyncInstalled},
		"skills sync mode: installed (update skills already in .agents/skills) | default | all | none | <skill-id>",
	)
	flags.StringVarP(
		&opt.ApplicationName,
		"appName",
		"a",
		"",
		"application name for templates (default: derived from module or directory)",
	)

	_ = CmdSetupSync.RegisterFlagCompletionFunc(
		"kind",
		func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{syncKindAuto, KindBackend, KindFullstack, KindFrontend}, cobra.ShellCompDirectiveNoFileComp
		},
	)

	_ = CmdSetupSync.RegisterFlagCompletionFunc(
		"only",
		func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			return []string{
				syncSectionAll,
				syncSectionAgents,
				syncSectionAI,
				syncSectionHooks,
				syncSectionDevops,
				syncSectionDocs,
				syncSectionSkills,
				syncSectionBackend,
				syncSectionFrontend,
			}, cobra.ShellCompDirectiveNoFileComp
		},
	)

	Cmd.AddCommand(CmdSetupSync)
}

var CmdSetupSync = &cobra.Command{
	Use:   "sync",
	Short: "Refresh scaffolded workspace boilerplate from the embedded skeleton",
	Long: `Update scaffold-managed files in an existing workspace (AGENTS.md, devops/, hooks,
skills, Makefiles, etc.) without touching application code under cmd/, feature/, or pkg/.

By default only missing scaffold files are added. Pass --overwrite to replace files that
already exist. Run from the repository root, or from backend/ in a fullstack workspace.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := cmd.ParseFlags(args); err != nil {
			return err
		}

		return runSync(cmd)
	},
}

type workspaceLayout struct {
	Kind     string
	RepoRoot string
	GoRoot   string
}

func runSync(cmd *cobra.Command) error {
	layout, err := resolveWorkspaceLayout(syncOpt.RepoDir, syncOpt.Kind)
	if err != nil {
		return err
	}

	sections, err := resolveSyncSections(syncOpt.Only, layout.Kind)
	if err != nil {
		return err
	}

	opt.Kind = layout.Kind
	opt.RepositoryRootDir = layout.RepoRoot

	if f := cmd.Flag("repoModule"); f != nil && f.Changed {
		// keep explicit --repoModule from parent persistent flags
	} else if layout.GoRoot != "" {
		module, err := detectGoModule(layout.GoRoot)
		if err != nil {
			return fmt.Errorf("detect repository module: %w", err)
		}

		opt.RepositoryGoModule = module
	} else if opt.RepositoryGoModule == "" || opt.RepositoryGoModule == "github.com/your/repo" {
		opt.RepositoryGoModule = "github.com/your/" + filepath.Base(layout.RepoRoot)
	}

	if f := cmd.Flag("appName"); f == nil || !f.Changed {
		opt.ApplicationName = appNameFromModule(opt.RepositoryGoModule)
		if opt.ApplicationName == "" || opt.ApplicationName == "repo" {
			opt.ApplicationName = appNameFromPath(layout.RepoRoot)
		}
	}

	skillIDs, err := resolveSyncSkills(layout.RepoRoot, syncOpt.SkillsMode, layout.Kind)
	if err != nil {
		return err
	}

	opt.resolvedSkills = skillIDs

	templateInput := TemplateInput{
		ApplicationName: opt.ApplicationName,
		RepositoryPath:  goModulePrefix(),
		PackagePath:     strings.Trim(opt.FeatureDir, "/"),
		PackageName:     opt.FeatureName,
		RonyKitPath:     "github.com/clubpay/ronykit",
		Kind:            layout.Kind,
		Skills:          selectedSkillInfos(skillIDs),
	}

	skipExisting := !syncOpt.Overwrite
	callback := func(filePath string, _ bool) {
		cmd.Println("sync:", filePath)
	}

	cmd.Printf("Syncing %s workspace at %s (overwrite=%v)\n", layout.Kind, layout.RepoRoot, syncOpt.Overwrite)
	cmd.Printf("Sections: %s\n", strings.Join(sections, ", "))

	for _, section := range sections {
		if err := syncSection(section, layout, templateInput, skipExisting, callback); err != nil {
			return fmt.Errorf("sync %s: %w", section, err)
		}
	}

	cmd.Println("Workspace sync complete")

	return nil
}

func resolveWorkspaceLayout(repoDir, kindFlag string) (workspaceLayout, error) {
	abs, err := filepath.Abs(repoDir)
	if err != nil {
		return workspaceLayout{}, err
	}

	detected, err := detectWorkspaceLayout(abs)
	if err != nil {
		return workspaceLayout{}, err
	}

	if kindFlag == "" || kindFlag == syncKindAuto {
		return detected, nil
	}

	switch kindFlag {
	case KindBackend, KindFullstack, KindFrontend:
	default:
		return workspaceLayout{}, fmt.Errorf(
			"invalid workspace kind %q: must be auto, %q, %q or %q",
			kindFlag, KindBackend, KindFullstack, KindFrontend,
		)
	}

	if kindFlag != detected.Kind {
		return workspaceLayout{}, fmt.Errorf(
			"workspace kind mismatch: detected %q at %s but --kind %q was requested",
			detected.Kind, abs, kindFlag,
		)
	}

	return detected, nil
}

func detectWorkspaceLayout(abs string) (workspaceLayout, error) {
	if isDir(filepath.Join(abs, backendDir)) && fileExists(filepath.Join(abs, backendDir, "go.work")) {
		return workspaceLayout{
			Kind:     KindFullstack,
			RepoRoot: abs,
			GoRoot:   filepath.Join(abs, backendDir),
		}, nil
	}

	if fileExists(filepath.Join(abs, "go.work")) {
		parent := filepath.Dir(abs)
		if isDir(filepath.Join(parent, backendDir)) && fileExists(filepath.Join(parent, backendDir, "go.work")) &&
			filepath.Base(abs) == backendDir {
			return workspaceLayout{
				Kind:     KindFullstack,
				RepoRoot: parent,
				GoRoot:   abs,
			}, nil
		}

		return workspaceLayout{
			Kind:     KindBackend,
			RepoRoot: abs,
			GoRoot:   abs,
		}, nil
	}

	if isDir(filepath.Join(abs, frontendDir)) {
		return workspaceLayout{
			Kind:     KindFrontend,
			RepoRoot: abs,
			GoRoot:   "",
		}, nil
	}

	return workspaceLayout{}, fmt.Errorf(
		"%s does not look like a ronyup workspace (expected go.work, backend/go.work, or frontend/)",
		abs,
	)
}

func resolveSyncSections(only []string, kind string) ([]string, error) {
	if len(only) == 0 {
		only = []string{syncSectionAll}
	}

	set := map[string]bool{}

	for _, raw := range only {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			if part == syncSectionAll {
				for _, s := range defaultSyncSections(kind) {
					set[s] = true
				}

				continue
			}

			if !isSyncSection(part) {
				return nil, fmt.Errorf("unknown sync section %q", part)
			}

			if part == syncSectionBackend && !hasBackend(kind) {
				continue
			}

			if part == syncSectionFrontend && !hasFrontend(kind) {
				continue
			}

			set[part] = true
		}
	}

	if len(set) == 0 {
		return nil, fmt.Errorf("no sync sections selected")
	}

	order := defaultSyncSections(kind)

	var sections []string

	for _, s := range order {
		if set[s] {
			sections = append(sections, s)
		}
	}

	return sections, nil
}

func defaultSyncSections(kind string) []string {
	sections := []string{
		syncSectionAgents,
		syncSectionAI,
		syncSectionHooks,
		syncSectionDevops,
		syncSectionDocs,
		syncSectionSkills,
	}

	if hasBackend(kind) {
		sections = append(sections, syncSectionBackend)
	}

	if hasFrontend(kind) {
		sections = append(sections, syncSectionFrontend)
	}

	return sections
}

func isSyncSection(name string) bool {
	switch name {
	case syncSectionAgents, syncSectionAI, syncSectionHooks, syncSectionDevops,
		syncSectionDocs, syncSectionSkills, syncSectionBackend, syncSectionFrontend, syncSectionAll:
		return true
	default:
		return false
	}
}

func syncSection(
	section string,
	layout workspaceLayout,
	templateInput TemplateInput,
	skipExisting bool,
	callback func(string, bool),
) error {
	switch section {
	case syncSectionAgents:
		return syncWorkspacePaths(layout, templateInput, skipExisting, callback, "AGENTS.mdtmpl")
	case syncSectionAI:
		return syncWorkspacePaths(layout, templateInput, skipExisting, callback, ".ai", ".cursor/mcp.json")
	case syncSectionHooks:
		return syncWorkspacePaths(layout, templateInput, skipExisting, callback, ".cursor/hooks", ".cursor/hooks.jsontmpl")
	case syncSectionDevops:
		return syncWorkspacePaths(layout, templateInput, skipExisting, callback, "devops")
	case syncSectionDocs:
		return syncWorkspacePaths(layout, templateInput, skipExisting, callback, "docs/design/README.MD")
	case syncSectionSkills:
		return syncSkills(layout, templateInput, skipExisting, callback)
	case syncSectionBackend:
		return syncBackendBoilerplate(layout, templateInput, skipExisting, callback)
	case syncSectionFrontend:
		return syncFrontendBoilerplate(layout, templateInput, skipExisting, callback)
	default:
		return fmt.Errorf("unsupported section %q", section)
	}
}

func syncWorkspacePaths(
	layout workspaceLayout,
	templateInput TemplateInput,
	skipExisting bool,
	callback func(string, bool),
	allowed ...string,
) error {
	allowedSet := make(map[string]bool, len(allowed))
	for _, p := range allowed {
		allowedSet[p] = true
	}

	return z.CopyDir(z.CopyDirParams{
		FS:             internal.Skeleton,
		SrcPathPrefix:  filepath.Join("skeleton", "workspace"),
		DestPathPrefix: layout.RepoRoot,
		TemplateInput:  templateInput,
		SkipExisting:   skipExisting,
		DestMapper:     workspacePathFilter(layout.RepoRoot, layout.Kind, allowedSet),
		Callback:       callback,
	})
}

func workspacePathFilter(repoRoot, kind string, allowed map[string]bool) func(string) (string, bool) {
	return func(relPath string) (string, bool) {
		rel := filepath.ToSlash(relPath)

		if kind == KindFrontend && strings.HasSuffix(rel, "hooks/backend-verify.sh") {
			return "", true
		}

		if !pathAllowed(rel, allowed) {
			return "", true
		}

		return filepath.Join(repoRoot, relPath), false
	}
}

func syncSkills(
	layout workspaceLayout,
	templateInput TemplateInput,
	skipExisting bool,
	callback func(string, bool),
) error {
	if err := syncWorkspacePaths(
		layout,
		templateInput,
		skipExisting,
		callback,
		".agents/skills/ronykit-framework",
	); err != nil {
		return err
	}

	dest := filepath.Join(layout.RepoRoot, ".agents", "skills")

	for _, id := range opt.resolvedSkills {
		if id == "ronykit-framework" || !skillExists(id) {
			continue
		}

		err := z.CopyDir(z.CopyDirParams{
			FS:             internal.Skeleton,
			SrcPathPrefix:  filepath.ToSlash(filepath.Join(skillsSrcPrefix, id)),
			DestPathPrefix: filepath.Join(dest, id),
			SkipExisting:   skipExisting,
			Callback:       callback,
		})
		if err != nil {
			return fmt.Errorf("skill %q: %w", id, err)
		}
	}

	return nil
}

func syncBackendBoilerplate(
	layout workspaceLayout,
	templateInput TemplateInput,
	skipExisting bool,
	callback func(string, bool),
) error {
	allowed := map[string]bool{
		"Makefile":          true,
		"verify.sh":         true,
		".golangci.yml":     true,
		"feature/README.MD": true,
	}

	dest := backendDestPrefix(layout.RepoRoot)

	return z.CopyDir(z.CopyDirParams{
		FS:             internal.Skeleton,
		SrcPathPrefix:  filepath.Join("skeleton", "backend"),
		DestPathPrefix: dest,
		TemplateInput:  templateInput,
		SkipExisting:   skipExisting,
		DestMapper:     prefixFilter(dest, allowed),
		Callback:       callback,
	})
}

func syncFrontendBoilerplate(
	layout workspaceLayout,
	templateInput TemplateInput,
	skipExisting bool,
	callback func(string, bool),
) error {
	allowed := map[string]bool{
		"Makefile":  true,
		"verify.sh": true,
		"README.MD": true,
	}

	dest := filepath.Join(layout.RepoRoot, frontendDir)

	return z.CopyDir(z.CopyDirParams{
		FS:             internal.Skeleton,
		SrcPathPrefix:  filepath.Join("skeleton", "frontend"),
		DestPathPrefix: dest,
		TemplateInput:  templateInput,
		SkipExisting:   skipExisting,
		DestMapper:     prefixFilter(dest, allowed),
		Callback:       callback,
	})
}

func prefixFilter(base string, allowed map[string]bool) func(string) (string, bool) {
	return func(relPath string) (string, bool) {
		rel := filepath.ToSlash(relPath)
		if !pathAllowed(rel, allowed) {
			return "", true
		}

		return filepath.Join(base, relPath), false
	}
}

func pathAllowed(rel string, allowed map[string]bool) bool {
	for prefix := range allowed {
		if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
			return true
		}
	}

	return false
}

func resolveSyncSkills(repoRoot string, modes []string, kind string) ([]string, error) {
	if len(modes) == 0 {
		modes = []string{skillSyncInstalled}
	}

	installed := listInstalledSkillIDs(repoRoot)
	set := map[string]bool{}

	for _, raw := range modes {
		for _, token := range strings.Split(raw, ",") {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}

			switch token {
			case skillSyncInstalled:
				for _, id := range installed {
					if skillExists(id) {
						set[id] = true
					}
				}
			case skillTokenDefault, skillTokenDefaults:
				for _, id := range defaultSkillIDs(kind) {
					set[id] = true
				}
			case skillTokenAll:
				for _, id := range allSkillIDs() {
					set[id] = true
				}
			case skillTokenNone:
				set = map[string]bool{}
			default:
				if !skillExists(token) {
					return nil, fmt.Errorf("unknown skill %q", token)
				}

				set[token] = true
			}
		}
	}

	return filterCatalogOrder(set), nil
}

func listInstalledSkillIDs(repoRoot string) []string {
	entries, err := os.ReadDir(filepath.Join(repoRoot, ".agents", "skills"))
	if err != nil {
		return nil
	}

	var ids []string

	for _, e := range entries {
		if e.IsDir() && e.Name() != "ronykit-framework" {
			ids = append(ids, e.Name())
		}
	}

	return ids
}

func appNameFromModule(module string) string {
	module = strings.TrimSuffix(module, "/")
	if module == "" {
		return ""
	}

	seg := module
	if i := strings.LastIndex(module, "/"); i >= 0 {
		seg = module[i+1:]
	}

	return strings.NewReplacer("_", "-").Replace(seg)
}

func appNameFromPath(dir string) string {
	return strings.NewReplacer("_", "-").Replace(filepath.Base(dir))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}
