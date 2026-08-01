package scaffold

import (
	"context"
	"path/filepath"

	appscaffold "github.com/clubpay/ronykit/ronyup/internal/scaffold"
	"github.com/clubpay/ronykit/x/rkit"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// JSON schema keys shared by tool input schema definitions.
const (
	schemaKeyType        = "type"
	schemaKeyObject      = "object"
	schemaKeyProperties  = "properties"
	schemaKeyItems       = "items"
	schemaKeyRequired    = "required"
	schemaKeyEnum        = "enum"
	schemaKeyDefault     = "default"
	schemaKeyDescription = "description"
	schemaTypeString     = "string"
	schemaTypeBoolean    = "boolean"
	schemaTypeArray      = "array"
	argWorkspacePath     = "workspacePath"
	argName              = "name"
	argDescription       = "description"
)

type syncWorkspaceArgs struct {
	WorkspacePath string   `json:"workspacePath"`
	Kind          string   `json:"kind"`
	Only          []string `json:"only"`
	Overwrite     bool     `json:"overwrite"`
	Skills        []string `json:"skills"`
}

type setupBundleArgs struct {
	WorkspacePath string   `json:"workspacePath"`
	Name          string   `json:"name"`
	Services      []string `json:"services"`
	Description   string   `json:"description"`
	Gen           bool     `json:"gen"`
	Remove        bool     `json:"remove"`
}

type migrateBundlesArgs struct {
	WorkspacePath string `json:"workspacePath"`
	DryRun        bool   `json:"dryRun"`
}

// registerLifecycle registers the sync_workspace, setup_bundle and
// migrate_bundles tools that call internal/scaffold in-process.
func registerLifecycle(srv *mcpsdk.Server) {
	registerSyncWorkspace(srv)
	registerSetupBundle(srv)
	registerMigrateBundles(srv)
}

func stringArg(description string) map[string]any {
	return map[string]any{
		schemaKeyType:        schemaTypeString,
		schemaKeyDescription: description,
	}
}

func stringSliceArg(description string) map[string]any {
	return map[string]any{
		schemaKeyType:        schemaTypeArray,
		schemaKeyItems:       map[string]any{schemaKeyType: schemaTypeString},
		schemaKeyDescription: description,
	}
}

func boolArg(description string) map[string]any {
	return map[string]any{
		schemaKeyType:        schemaTypeBoolean,
		schemaKeyDescription: description,
		schemaKeyDefault:     false,
	}
}

func registerSyncWorkspace(srv *mcpsdk.Server) {
	tool := &mcpsdk.Tool{
		Name: "sync_workspace",
		Description: "Refresh scaffold-managed boilerplate (AGENTS.md, devops/, hooks, " +
			"skills, Makefiles) in an existing workspace from the embedded skeleton, " +
			"without touching application code under cmd/, feature/, or pkg/. " +
			"Does NOT perform the bundle layout migration — use migrate_bundles for that.",
		InputSchema: map[string]any{
			schemaKeyType: schemaKeyObject,
			schemaKeyProperties: map[string]any{
				argWorkspacePath: stringArg(
					"The path to the existing workspace (repository root, or backend/ " +
						"in a fullstack workspace).",
				),
				"kind": map[string]any{
					schemaKeyType:        schemaTypeString,
					schemaKeyDescription: "Workspace kind override. Omit for auto-detection.",
					schemaKeyEnum: []string{
						"auto",
						appscaffold.KindBackend,
						appscaffold.KindFullstack,
						appscaffold.KindFrontend,
					},
					schemaKeyDefault: "auto",
				},
				"only": stringSliceArg(
					"Sections to refresh: agents | ai | hooks | devops | docs | skills | " +
						"backend | frontend | all. Omit to refresh every section applicable to the " +
						"detected kind.",
				),
				"overwrite": boolArg(
					"Replace existing scaffold files. Default: add missing files only.",
				),
				"skills": stringSliceArg(
					"Skills sync mode for the skills section: installed (default), " +
						"default, all, none, or explicit skill IDs.",
				),
			},
			schemaKeyRequired: []string{argWorkspacePath},
		},
	}

	mcpsdk.AddTool(srv, tool,
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, args syncWorkspaceArgs) (
			*mcpsdk.CallToolResult, any, error,
		) {
			if args.WorkspacePath == "" {
				return errorResult(rkit.L("workspacePath is required")), nil, nil
			}

			startDir, err := filepath.Abs(args.WorkspacePath)
			if err != nil {
				return errorResult(rkit.L("invalid workspacePath %s: %v", args.WorkspacePath, err)), nil, nil
			}

			log := &appscaffold.BufferLogger{}

			err = appscaffold.SyncWorkspace(appscaffold.SyncRequest{
				StartDir:   startDir,
				Kind:       args.Kind,
				Only:       args.Only,
				Overwrite:  args.Overwrite,
				SkillsMode: args.Skills,
			}, log)
			if err != nil {
				return errorResult(
					rkit.L("failed to sync workspace: %v", err),
					rkit.L("%s", log.String()),
				), nil, nil
			}

			return textResult(
				rkit.L("Workspace at %s synced.", startDir),
				rkit.L("%s", log.String()),
			), nil, nil
		})
}

func registerSetupBundle(srv *mcpsdk.Server) {
	tool := &mcpsdk.Tool{
		Name: "setup_bundle",
		Description: "Create or refresh executable bundles under cmd/<name>/ that compile in " +
			"only the selected feature modules. The workspace must already use the bundle layout " +
			"(run migrate_bundles first on legacy workspaces).",
		InputSchema: map[string]any{
			schemaKeyType: schemaKeyObject,
			schemaKeyProperties: map[string]any{
				argWorkspacePath: stringArg(
					"The path to the existing workspace (repository root or Go workspace root).",
				),
				argName: stringArg(
					"Bundle name (cmd/<name>/ directory). Required unless gen is true.",
				),
				"services": stringSliceArg(
					"Feature module paths (settings.ModuleName) to compile in, or " +
						"[\"*\"] for all features. Required when creating a bundle.",
				),
				argDescription: stringArg(
					"Optional description stored in bundles.yaml.",
				),
				"gen": boolArg(
					"Regenerate features.go for every bundle from bundles.yaml and exit.",
				),
				"remove": boolArg(
					"Remove the named bundle from bundles.yaml and delete cmd/<name>/.",
				),
			},
			schemaKeyRequired: []string{argWorkspacePath},
		},
	}

	mcpsdk.AddTool(srv, tool,
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, args setupBundleArgs) (
			*mcpsdk.CallToolResult, any, error,
		) {
			if args.WorkspacePath == "" {
				return errorResult(rkit.L("workspacePath is required")), nil, nil
			}

			startDir, err := filepath.Abs(args.WorkspacePath)
			if err != nil {
				return errorResult(rkit.L("invalid workspacePath %s: %v", args.WorkspacePath, err)), nil, nil
			}

			log := &appscaffold.BufferLogger{}

			switch {
			case args.Gen:
				err = appscaffold.RegenerateBundleFeatures(ctx, appscaffold.RegenerateBundlesRequest{
					StartDir: startDir,
				}, log)
			case args.Remove:
				err = appscaffold.RemoveBundle(ctx, appscaffold.BundleRequest{
					StartDir: startDir,
					Name:     args.Name,
				}, log)
			default:
				err = appscaffold.CreateBundle(ctx, appscaffold.BundleRequest{
					StartDir:    startDir,
					Name:        args.Name,
					Services:    args.Services,
					Description: args.Description,
				}, log)
			}

			if err != nil {
				return errorResult(
					rkit.L("failed to setup bundle: %v", err),
					rkit.L("%s", log.String()),
				), nil, nil
			}

			return textResult(
				rkit.L("Bundle operation completed in workspace %s.", startDir),
				rkit.L("%s", log.String()),
			), nil, nil
		})
}

func registerMigrateBundles(srv *mcpsdk.Server) {
	tool := &mcpsdk.Tool{
		Name: "migrate_bundles",
		Description: "Upgrade a workspace created before executable bundles to the current " +
			"layout (pkg/runner, bundles.yaml, thin cmd/all-in-one/main.go). Idempotent: safe " +
			"to run multiple times. Use dryRun to preview the planned steps first.",
		InputSchema: map[string]any{
			schemaKeyType: schemaKeyObject,
			schemaKeyProperties: map[string]any{
				argWorkspacePath: stringArg(
					"The path to the existing workspace (Go workspace root, or " +
						"repository root in a fullstack workspace).",
				),
				"dryRun": boolArg(
					"Print the planned changes without writing files.",
				),
			},
			schemaKeyRequired: []string{argWorkspacePath},
		},
	}

	mcpsdk.AddTool(srv, tool,
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, args migrateBundlesArgs) (
			*mcpsdk.CallToolResult, any, error,
		) {
			if args.WorkspacePath == "" {
				return errorResult(rkit.L("workspacePath is required")), nil, nil
			}

			startDir, err := filepath.Abs(args.WorkspacePath)
			if err != nil {
				return errorResult(rkit.L("invalid workspacePath %s: %v", args.WorkspacePath, err)), nil, nil
			}

			log := &appscaffold.BufferLogger{}

			err = appscaffold.MigrateBundles(ctx, appscaffold.MigrateBundlesRequest{
				StartDir: startDir,
				DryRun:   args.DryRun,
			}, log)
			if err != nil {
				return errorResult(
					rkit.L("failed to migrate bundles: %v", err),
					rkit.L("%s", log.String()),
				), nil, nil
			}

			return textResult(
				rkit.L("Bundle migration completed in workspace %s.", startDir),
				rkit.L("%s", log.String()),
			), nil, nil
		})
}
