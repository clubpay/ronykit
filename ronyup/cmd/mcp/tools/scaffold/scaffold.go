package scaffold

import (
	"context"
	"path/filepath"

	appscaffold "github.com/clubpay/ronykit/ronyup/internal/scaffold"
	"github.com/clubpay/ronykit/x/rkit"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type workspaceArgs struct {
	Path   string   `json:"path"`
	Kind   string   `json:"kind"`
	Skills []string `json:"skills"`
}

type featureArgs struct {
	WorkspacePath   string `json:"workspacePath"`
	Name            string `json:"name"`
	Template        string `json:"template"`
	FeaturePrefix   string `json:"featurePrefix"`
	GroupByTemplate bool   `json:"groupByTemplate"`
	SkipDesignGate  bool   `json:"skipDesignGate"`
}

// Register registers scaffold MCP tools that call internal/scaffold in-process.
func Register(srv *mcpsdk.Server) {
	registerSetupWorkspace(srv)
	registerSetupFeature(srv)
	registerLifecycle(srv)
}

func registerSetupWorkspace(srv *mcpsdk.Server) {
	tool := &mcpsdk.Tool{
		Name:        "scaffold_workspace",
		Description: "Initialize a new ronykit workspace at the specified directory.",
		InputSchema: map[string]any{
			schemaKeyType: schemaKeyObject,
			schemaKeyProperties: map[string]any{
				"path": map[string]any{
					schemaKeyType:        schemaTypeString,
					schemaKeyDescription: "The absolute or relative path to initialize the workspace.",
				},
				"kind": map[string]any{
					schemaKeyType: schemaTypeString,
					schemaKeyDescription: "Workspace layout: 'backend' (Go-only at the root, default), " +
						"'fullstack' (backend/ + frontend/ split, with the Go workspace under backend/ " +
						"and devops/, docs/ and AI config kept at the root), or 'frontend' (frontend/ app " +
						"plus shared AI config and docs/ at the root, with no Go workspace).",
					schemaKeyDefault: "backend",
					"enum":           appscaffold.WorkspaceKinds,
				},
				"skills": map[string]any{
					schemaKeyType: schemaTypeArray,
					"items": map[string]any{
						schemaKeyType: schemaTypeString,
					},
					schemaKeyDescription: "Agent skills to pre-install into .agents/skills. Skill IDs or the " +
						"tokens 'default' (kind defaults), 'all', or 'none'. Omit to install the " +
						"defaults for the chosen kind.",
				},
			},
			schemaKeyRequired: []string{"path"},
		},
	}

	mcpsdk.AddTool(srv, tool,
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, args workspaceArgs) (
			*mcpsdk.CallToolResult, any, error,
		) {
			if args.Path == "" {
				return errorResult(rkit.L("path is required")), nil, nil
			}

			absPath, err := filepath.Abs(args.Path)
			if err != nil {
				return errorResult(rkit.L("invalid path %s: %v", args.Path, err)), nil, nil
			}

			log := &appscaffold.BufferLogger{}

			err = appscaffold.SetupWorkspace(ctx, appscaffold.WorkspaceRequest{
				Path:   absPath,
				Kind:   args.Kind,
				Skills: args.Skills,
			}, log)
			if err != nil {
				return errorResult(
					rkit.L("failed to setup workspace: %v", err),
					rkit.L("%s", log.String()),
				), nil, nil
			}

			return textResult(
				rkit.L("Workspace successfully setup at %s.", absPath),
				rkit.L("%s", log.String()),
			), nil, nil
		})
}

func registerSetupFeature(srv *mcpsdk.Server) {
	tool := &mcpsdk.Tool{
		Name: "scaffold_feature",
		Description: "Create a new feature in the current workspace. " +
			"GATED: requires an approved SRS (docs/design/<name>-srs.md) and SDD " +
			"(docs/design/<name>-sdd.md), each with frontmatter `status: approved`. " +
			"Fails with guidance if they are missing or unapproved. Use the write-srs " +
			"and write-sdd prompts first; only set skipDesignGate=true when the user " +
			"explicitly asks to skip the design documents.",
		InputSchema: map[string]any{
			schemaKeyType: schemaKeyObject,
			schemaKeyProperties: map[string]any{
				"workspacePath": map[string]any{
					schemaKeyType:        schemaTypeString,
					schemaKeyDescription: "The path to the existing workspace.",
				},
				"name": map[string]any{
					schemaKeyType:        schemaTypeString,
					schemaKeyDescription: "The name of the feature to create.",
				},
				"template": map[string]any{
					schemaKeyType:        schemaTypeString,
					schemaKeyDescription: "Feature template: service, job, or gateway.",
					schemaKeyDefault:     "service",
					"enum":               appscaffold.FeatureTemplates,
				},
				"featurePrefix": map[string]any{
					schemaKeyType:        schemaTypeString,
					schemaKeyDescription: "Parent directory for feature modules.",
					schemaKeyDefault:     appscaffold.DefaultFeaturePrefix,
				},
				"groupByTemplate": map[string]any{
					schemaKeyType:        schemaTypeBoolean,
					schemaKeyDescription: "When true, place the module under {featurePrefix}/{template}/{name}/.",
					schemaKeyDefault:     false,
				},
				"skipDesignGate": map[string]any{
					schemaKeyType: schemaTypeBoolean,
					schemaKeyDescription: "Bypass the approved SRS/SDD requirement. Only set to true " +
						"when the user explicitly asks to skip the design documents.",
					schemaKeyDefault: false,
				},
			},
			schemaKeyRequired: []string{"workspacePath", "name"},
		},
	}

	mcpsdk.AddTool(srv, tool,
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, args featureArgs) (
			*mcpsdk.CallToolResult, any, error,
		) {
			if args.WorkspacePath == "" {
				return errorResult(rkit.L("workspacePath is required")), nil, nil
			}

			if args.Name == "" {
				return errorResult(rkit.L("name is required")), nil, nil
			}

			if !args.SkipDesignGate {
				if problems := checkDesignGate(args.WorkspacePath, args.Name); len(problems) > 0 {
					return designGateError(args.Name, problems), nil, nil
				}
			}

			startDir, err := filepath.Abs(args.WorkspacePath)
			if err != nil {
				return errorResult(rkit.L("invalid workspacePath %s: %v", args.WorkspacePath, err)), nil, nil
			}

			log := &appscaffold.BufferLogger{}

			err = appscaffold.SetupFeature(ctx, appscaffold.FeatureRequest{
				StartDir:        startDir,
				FeatureDir:      args.Name,
				FeatureName:     args.Name,
				Template:        args.Template,
				FeaturePrefix:   args.FeaturePrefix,
				GroupByTemplate: args.GroupByTemplate,
			}, log)
			if err != nil {
				return errorResult(
					rkit.L("failed to setup feature: %v", err),
					rkit.L("%s", log.String()),
				), nil, nil
			}

			return textResult(
				rkit.L("Feature '%s' successfully created in workspace %s.", args.Name, startDir),
				rkit.L("%s", log.String()),
			), nil, nil
		})
}

func errorResult(lines ...rkit.StrLine) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		IsError: true,
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{
				Text: rkit.StrLines(lines...),
			},
		},
	}
}

func textResult(lines ...rkit.StrLine) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{
				Text: rkit.StrLines(lines...),
			},
		},
	}
}
