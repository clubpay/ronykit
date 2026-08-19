package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/ronyup/internal/z"
)

// Sync section identifiers for SyncRequest.Only.
const (
	SyncSectionAgents   = "agents"
	SyncSectionAI       = "ai"
	SyncSectionHooks    = "hooks"
	SyncSectionDevops   = "devops"
	SyncSectionDocs     = "docs"
	SyncSectionSkills   = "skills"
	SyncSectionBackend  = "backend"
	SyncSectionFrontend = "frontend"
	SyncSectionAll      = "all"
	// SkillSyncInstalled updates only skills already present in .agents/skills.
	SkillSyncInstalled = "installed"
)

// SyncSections lists every selectable sync section in default order.
var SyncSections = []string{
	SyncSectionAgents,
	SyncSectionAI,
	SyncSectionHooks,
	SyncSectionDevops,
	SyncSectionDocs,
	SyncSectionSkills,
	SyncSectionBackend,
	SyncSectionFrontend,
}

// SyncRequest configures SyncWorkspace.
type SyncRequest struct {
	// StartDir is the repository root to sync (layout is auto-detected).
	StartDir string
	// Kind overrides detection: auto (default) | backend | fullstack | frontend.
	Kind string
	// Only selects sections to refresh (default: all applicable to the kind).
	Only []string
	// Overwrite replaces existing scaffold files (default: add missing only).
	Overwrite bool
	// SkillsMode selects skills to refresh: installed (default) | default |
	// all | none | <skill-id> (comma separated tokens allowed).
	SkillsMode []string
	// Module overrides the Go module path (default: detected from go.work).
	Module string
	// AppName overrides the application name (default: derived from module or path).
	AppName string
}

// SyncWorkspace refreshes scaffold-managed boilerplate in an existing
// workspace without touching application code under cmd/, feature/, or pkg/.
func SyncWorkspace(req SyncRequest, log Logger) error {
	if log == nil {
		log = DiscardLogger{}
	}

	layout, err := ResolveWorkspaceLayout(req.StartDir, req.Kind)
	if err != nil {
		return err
	}

	sections, err := resolveSyncSections(req.Only, layout.Kind)
	if err != nil {
		return err
	}

	module := req.Module
	if module == "" {
		if layout.GoRoot != "" {
			module, err = detectGoModule(layout.GoRoot)
			if err != nil {
				return fmt.Errorf("detect repository module: %w", err)
			}
		} else {
			module = "github.com/your/" + filepath.Base(layout.RepoRoot)
		}
	}

	appName := req.AppName
	if appName == "" {
		appName = appNameFromModule(module)
		if appName == "" || appName == "repo" {
			appName = appNameFromPath(layout.RepoRoot)
		}
	}

	skillIDs, err := resolveSyncSkills(layout.RepoRoot, req.SkillsMode, layout.Kind)
	if err != nil {
		return err
	}

	templateInput := TemplateInput{
		ApplicationName: appName,
		RepositoryPath:  goModulePrefix(module, layout.Kind),
		RonyKitPath:     RonyKitModulePath,
		Kind:            layout.Kind,
		Skills:          selectedSkillInfos(skillIDs),
	}

	skipExisting := !req.Overwrite
	callback := func(filePath string, _ bool) {
		log.Println("sync:", filePath)
	}

	log.Printf("Syncing %s workspace at %s (overwrite=%v)\n", layout.Kind, layout.RepoRoot, req.Overwrite)
	log.Printf("Sections: %s\n", strings.Join(sections, ", "))

	for _, section := range sections {
		if err := syncSection(section, layout, templateInput, skillIDs, skipExisting, callback); err != nil {
			return fmt.Errorf("sync %s: %w", section, err)
		}
	}

	log.Println("Workspace sync complete")

	return nil
}

func resolveSyncSections(only []string, kind string) ([]string, error) {
	if len(only) == 0 {
		only = []string{SyncSectionAll}
	}

	set := map[string]bool{}

	for _, raw := range only {
		for part := range strings.SplitSeq(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			if part == SyncSectionAll {
				for _, s := range defaultSyncSections(kind) {
					set[s] = true
				}

				continue
			}

			if !isSyncSection(part) {
				return nil, fmt.Errorf("unknown sync section %q", part)
			}

			if part == SyncSectionBackend && !HasBackend(kind) {
				continue
			}

			if part == SyncSectionFrontend && !HasFrontend(kind) {
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
		SyncSectionAgents,
		SyncSectionAI,
		SyncSectionHooks,
		SyncSectionDevops,
		SyncSectionDocs,
		SyncSectionSkills,
	}

	if HasBackend(kind) {
		sections = append(sections, SyncSectionBackend)
	}

	if HasFrontend(kind) {
		sections = append(sections, SyncSectionFrontend)
	}

	return sections
}

func isSyncSection(name string) bool {
	switch name {
	case SyncSectionAgents, SyncSectionAI, SyncSectionHooks, SyncSectionDevops,
		SyncSectionDocs, SyncSectionSkills, SyncSectionBackend, SyncSectionFrontend, SyncSectionAll:
		return true
	default:
		return false
	}
}

func syncSection(
	section string,
	layout WorkspaceLayout,
	templateInput TemplateInput,
	skillIDs []string,
	skipExisting bool,
	callback func(string, bool),
) error {
	switch section {
	case SyncSectionAgents:
		return syncWorkspacePaths(
			layout,
			templateInput,
			skipExisting,
			callback,
			"AGENTS.mdtmpl",
			"CLAUDE.md",
			".cursorignore",
			".claude/settings.json",
			".editorconfig",
			"CODEOWNERS",
		)
	case SyncSectionAI:
		return syncWorkspacePaths(layout, templateInput, skipExisting, callback, ".ai", ".cursor/mcp.json")
	case SyncSectionHooks:
		return syncWorkspacePaths(
			layout,
			templateInput,
			skipExisting,
			callback,
			".cursor/hooks",
			".cursor/hooks.jsontmpl",
		)
	case SyncSectionDevops:
		return syncWorkspacePaths(layout, templateInput, skipExisting, callback, "devops")
	case SyncSectionDocs:
		return syncWorkspacePaths(layout, templateInput, skipExisting, callback, "docs/design/README.MD")
	case SyncSectionSkills:
		return syncSkills(layout, templateInput, skillIDs, skipExisting, callback)
	case SyncSectionBackend:
		return syncBackendBoilerplate(layout, templateInput, skipExisting, callback)
	case SyncSectionFrontend:
		return syncFrontendBoilerplate(layout, templateInput, skipExisting, callback)
	default:
		return fmt.Errorf("unsupported section %q", section)
	}
}

func syncWorkspacePaths(
	layout WorkspaceLayout,
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
	layout WorkspaceLayout,
	templateInput TemplateInput,
	skillIDs []string,
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

	for _, id := range skillIDs {
		if id == "ronykit-framework" || !SkillExists(id) {
			continue
		}

		err := z.CopyDir(z.CopyDirParams{
			FS:             internal.Skeleton,
			SrcPathPrefix:  filepath.ToSlash(filepath.Join(SkillsSrcPrefix, id)),
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
	layout WorkspaceLayout,
	templateInput TemplateInput,
	skipExisting bool,
	callback func(string, bool),
) error {
	allowed := map[string]bool{
		"Makefile":          true,
		"verify.sh":         true,
		".golangci.yml":     true,
		"bundles.yaml":      true,
		"feature/README.MD": true,
	}

	dest := BackendDestPrefix(layout.RepoRoot, layout.Kind)

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
	layout WorkspaceLayout,
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
		modes = []string{SkillSyncInstalled}
	}

	installed := listInstalledSkillIDs(repoRoot)
	set := map[string]bool{}

	for _, raw := range modes {
		for token := range strings.SplitSeq(raw, ",") {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}

			switch token {
			case SkillSyncInstalled:
				for _, id := range installed {
					if SkillExists(id) {
						set[id] = true
					}
				}
			case SkillTokenDefault, SkillTokenDefaults:
				for _, id := range DefaultSkillIDs(kind) {
					set[id] = true
				}
			case SkillTokenAll:
				for _, id := range AllSkillIDs() {
					set[id] = true
				}
			case SkillTokenNone:
				set = map[string]bool{}
			default:
				if !SkillExists(token) {
					return nil, fmt.Errorf("unknown skill %q", token)
				}

				set[token] = true
			}
		}
	}

	return FilterCatalogOrder(set), nil
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
