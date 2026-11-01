package scaffold

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

func detectGoModule(workspaceDir string) (string, error) {
	goWorkPath := filepath.Join(workspaceDir, "go.work")

	data, err := os.ReadFile(goWorkPath)
	if err != nil {
		return "", fmt.Errorf("could not read go.work: %w", err)
	}

	work, err := modfile.ParseWork(goWorkPath, data, nil)
	if err != nil {
		return "", fmt.Errorf("could not parse go.work: %w", err)
	}

	if len(work.Use) == 0 {
		return "", fmt.Errorf("no use directive found in go.work")
	}

	for _, use := range work.Use {
		goModPath := filepath.Join(workspaceDir, filepath.FromSlash(use.Path), "go.mod")

		modData, err := os.ReadFile(goModPath)
		if err != nil {
			continue
		}

		mod, err := modfile.ParseLax(goModPath, modData, nil)
		if err != nil || mod.Module == nil || mod.Module.Mod.Path == "" {
			continue
		}

		modulePath := mod.Module.Mod.Path

		relDir := path.Clean(strings.TrimPrefix(filepath.ToSlash(use.Path), "./"))
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
