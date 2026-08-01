package scaffold

import (
	"fmt"
	"path/filepath"
)

// SyncKindAuto lets SyncWorkspace auto-detect the workspace kind.
const SyncKindAuto = "auto"

// WorkspaceLayout describes where the repository root and (optional) Go
// workspace root live for a scaffolded workspace.
type WorkspaceLayout struct {
	// Kind is backend | fullstack | frontend.
	Kind string
	// RepoRoot is the repository root (workspace skeleton destination).
	RepoRoot string
	// GoRoot is the directory containing go.work (empty for frontend kind).
	GoRoot string
}

// ResolveWorkspaceLayout detects the workspace layout under repoDir and
// validates it against kindFlag when kindFlag is a concrete kind (not auto).
func ResolveWorkspaceLayout(repoDir, kindFlag string) (WorkspaceLayout, error) {
	abs, err := filepath.Abs(repoDir)
	if err != nil {
		return WorkspaceLayout{}, err
	}

	detected, err := detectWorkspaceLayout(abs)
	if err != nil {
		return WorkspaceLayout{}, err
	}

	if kindFlag == "" || kindFlag == SyncKindAuto {
		return detected, nil
	}

	switch kindFlag {
	case KindBackend, KindFullstack, KindFrontend:
	default:
		return WorkspaceLayout{}, fmt.Errorf(
			"invalid workspace kind %q: must be auto, %q, %q or %q",
			kindFlag, KindBackend, KindFullstack, KindFrontend,
		)
	}

	if kindFlag != detected.Kind {
		return WorkspaceLayout{}, fmt.Errorf(
			"workspace kind mismatch: detected %q at %s but --kind %q was requested",
			detected.Kind, abs, kindFlag,
		)
	}

	return detected, nil
}

func detectWorkspaceLayout(abs string) (WorkspaceLayout, error) {
	if IsDir(filepath.Join(abs, backendDir)) && FileExists(filepath.Join(abs, backendDir, "go.work")) {
		return WorkspaceLayout{
			Kind:     KindFullstack,
			RepoRoot: abs,
			GoRoot:   filepath.Join(abs, backendDir),
		}, nil
	}

	if FileExists(filepath.Join(abs, "go.work")) {
		parent := filepath.Dir(abs)
		if IsDir(filepath.Join(parent, backendDir)) &&
			FileExists(filepath.Join(parent, backendDir, "go.work")) &&
			filepath.Base(abs) == backendDir {
			return WorkspaceLayout{
				Kind:     KindFullstack,
				RepoRoot: parent,
				GoRoot:   abs,
			}, nil
		}

		return WorkspaceLayout{
			Kind:     KindBackend,
			RepoRoot: abs,
			GoRoot:   abs,
		}, nil
	}

	if IsDir(filepath.Join(abs, frontendDir)) {
		return WorkspaceLayout{
			Kind:     KindFrontend,
			RepoRoot: abs,
			GoRoot:   "",
		}, nil
	}

	return WorkspaceLayout{}, fmt.Errorf(
		"%s does not look like a ronyup workspace (expected go.work, backend/go.work, or frontend/)",
		abs,
	)
}
