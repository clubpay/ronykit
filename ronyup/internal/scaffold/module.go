package scaffold

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// DetectGoModule reads go.work to find use directives, then reads the first
// resolvable go.mod to derive the repository root module path.
func DetectGoModule(workspaceDir string) (string, error) {
	return detectGoModule(workspaceDir)
}

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
