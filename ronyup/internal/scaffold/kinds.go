package scaffold

// Workspace kinds supported by setup workspace / scaffold_workspace.
const (
	// KindBackend scaffolds a Go-only workspace at the repository root.
	KindBackend = "backend"
	// KindFullstack scaffolds a backend/ + frontend/ split: the Go workspace
	// is under backend/ while AI config, devops/ and docs/ stay at root.
	KindFullstack = "fullstack"
	// KindFrontend scaffolds a frontend-only workspace: a frontend/ application
	// plus shared AI config and docs/ at the root, with no Go workspace.
	KindFrontend = "frontend"
)

// backendDir is the subdirectory that holds the Go workspace for fullstack scaffolds.
const backendDir = "backend"

// frontendDir is the subdirectory that holds the frontend application.
const frontendDir = "frontend"

// HasBackend reports whether the workspace kind includes a Go backend.
func HasBackend(kind string) bool { return kind != KindFrontend }

// HasFrontend reports whether the workspace kind includes a frontend app.
func HasFrontend(kind string) bool { return kind != KindBackend }

func hasBackend(kind string) bool  { return HasBackend(kind) }
func hasFrontend(kind string) bool { return HasFrontend(kind) }
