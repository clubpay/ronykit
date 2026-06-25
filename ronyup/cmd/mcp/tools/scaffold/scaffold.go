package scaffold

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/clubpay/ronykit/x/rkit"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Runner abstracts the command execution
type Runner interface {
	Run(ctx context.Context, cwd, name string, args ...string) (stdout, stderr string, err error)
}

// Register registers all scaffold-related tools to the given MCP server.
func Register(srv *mcpsdk.Server, runner Runner, executable string) {
	registerSetupWorkspace(srv, runner, executable)
	registerSetupFeature(srv, runner, executable)
}

func registerSetupWorkspace(srv *mcpsdk.Server, runner Runner, executable string) {
	tool := &mcpsdk.Tool{
		Name:        "scaffold_workspace",
		Description: "Initialize a new ronykit workspace at the specified directory.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "The absolute or relative path to initialize the workspace.",
				},
				"kind": map[string]any{
					"type": "string",
					"description": "Workspace layout: 'backend' (Go-only at the root, default), " +
						"'fullstack' (backend/ + frontend/ split, with the Go workspace under backend/ " +
						"and devops/, docs/ and AI config kept at the root), or 'frontend' (frontend/ app " +
						"plus shared AI config and docs/ at the root, with no Go workspace).",
					"default": "backend",
					"enum":    []string{"backend", "fullstack", "frontend"},
				},
				"skills": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "Agent skills to pre-install into .agents/skills. Skill IDs or the " +
						"tokens 'default' (kind defaults), 'all', or 'none'. Omit to install the " +
						"defaults for the chosen kind.",
				},
			},
			"required": []string{"path"},
		},
	}

	srv.AddTool(
		tool,
		func(ctx context.Context, request *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
			var args struct {
				Path   string   `json:"path"`
				Kind   string   `json:"kind"`
				Skills []string `json:"skills"`
			}
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return errorResult(rkit.L("failed to parse arguments: %v", err)), nil
			}

			if args.Path == "" {
				return errorResult(rkit.L("path is required")), nil
			}

			cliArgs := []string{"setup", "workspace"}
			if args.Kind != "" {
				cliArgs = append(cliArgs, "--kind", args.Kind)
			}

			if len(args.Skills) > 0 {
				cliArgs = append(cliArgs, "--skills", strings.Join(args.Skills, ","))
			}

			stdout, stderr, err := runner.Run(ctx, args.Path, executable, cliArgs...)
			if err != nil {
				return errorResult(
					rkit.L("failed to setup workspace: %v", err),
					rkit.L("Stderr: %s", stderr),
				), nil
			}

			return textResult(
				rkit.L("Workspace successfully setup at %s.", args.Path),
				rkit.L("Stdout:"),
				rkit.L("%s", stdout),
			), nil
		})
}

func registerSetupFeature(srv *mcpsdk.Server, runner Runner, executable string) {
	tool := &mcpsdk.Tool{
		Name: "scaffold_feature",
		Description: "Create a new feature in the current workspace. " +
			"GATED: requires an approved SRS (docs/design/<name>-srs.md) and SDD " +
			"(docs/design/<name>-sdd.md), each with frontmatter `status: approved`. " +
			"Fails with guidance if they are missing or unapproved. Use the write-srs " +
			"and write-sdd prompts first; only set skipDesignGate=true when the user " +
			"explicitly asks to skip the design documents.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workspacePath": map[string]any{
					"type":        "string",
					"description": "The path to the existing workspace.",
				},
				"name": map[string]any{
					"type":        "string",
					"description": "The name of the feature to create.",
				},
				"template": map[string]any{
					"type":        "string",
					"description": "Feature template: service, job, or gateway.",
					"default":     "service",
					"enum":        []string{"service"},
				},
				"featurePrefix": map[string]any{
					"type":        "string",
					"description": "Parent directory for feature modules.",
					"default":     "feature",
				},
				"groupByTemplate": map[string]any{
					"type":        "boolean",
					"description": "When true, place the module under {featurePrefix}/{template}/{name}/.",
					"default":     false,
				},
				"skipDesignGate": map[string]any{
					"type": "boolean",
					"description": "Bypass the approved SRS/SDD requirement. Only set to true " +
						"when the user explicitly asks to skip the design documents.",
					"default": false,
				},
			},
			"required": []string{"workspacePath", "name"},
		},
	}

	srv.AddTool(
		tool,
		func(ctx context.Context, request *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
			var args struct {
				WorkspacePath   string `json:"workspacePath"`
				Name            string `json:"name"`
				Template        string `json:"template"`
				FeaturePrefix   string `json:"featurePrefix"`
				GroupByTemplate bool   `json:"groupByTemplate"`
				SkipDesignGate  bool   `json:"skipDesignGate"`
			}
			if err := json.Unmarshal(request.Params.Arguments, &args); err != nil {
				return errorResult(rkit.L("failed to parse arguments: %v", err)), nil
			}

			if args.WorkspacePath == "" {
				return errorResult(rkit.L("workspacePath is required")), nil
			}

			if args.Name == "" {
				return errorResult(rkit.L("name is required")), nil
			}

			if !args.SkipDesignGate {
				if problems := checkDesignGate(args.WorkspacePath, args.Name); len(problems) > 0 {
					return designGateError(args.Name, problems), nil
				}
			}

			if args.Template == "" {
				args.Template = "service"
			}

			if args.FeaturePrefix == "" {
				args.FeaturePrefix = "feature"
			}

			cliArgs := []string{
				"setup", "feature",
				"--featureDir", args.Name,
				"--featureName", args.Name,
				"--template", args.Template,
				"--featurePrefix", args.FeaturePrefix,
			}
			if args.GroupByTemplate {
				cliArgs = append(cliArgs, "--groupByTemplate")
			}

			stdout, stderr, err := runner.Run(ctx, args.WorkspacePath, executable, cliArgs...)
			if err != nil {
				return errorResult(
					rkit.L("failed to setup feature: %v", err),
					rkit.L("Stderr: %s", stderr),
				), nil
			}

			return textResult(
				rkit.L("Feature '%s' successfully created in workspace %s.", args.Name, args.WorkspacePath),
				rkit.L("Stdout:"),
				rkit.L("%s", stdout),
			), nil
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
