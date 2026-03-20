package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

//nolint:lll // Keep explicit jsonschema descriptions for MCP clients.
type createWorkspaceInput struct {
	RepoDir    string            `json:"repo_dir"         jsonschema:"required,description:Destination directory for the new workspace"`
	RepoModule string            `json:"repo_module"      jsonschema:"required,description:Go module path for the repository"`
	Force      bool              `json:"force,omitempty"  jsonschema:"description:Clean destination directory before setup"`
	Custom     map[string]string `json:"custom,omitempty" jsonschema:"description:Custom key/value replacements for templates"`
}

type createWorkspaceOutput struct {
	RepoDir string `json:"repo_dir"`
	Stdout  string `json:"stdout,omitempty"`
	Stderr  string `json:"stderr,omitempty"`
}

//nolint:lll // Keep explicit jsonschema descriptions for MCP clients.
type createFeatureInput struct {
	WorkspaceDir string            `json:"workspace_dir"      jsonschema:"required,description:Workspace root directory that contains go.work"`
	RepoModule   string            `json:"repo_module"        jsonschema:"required,description:Go module path for the repository"`
	FeatureDir   string            `json:"feature_dir"        jsonschema:"required,description:Feature directory relative to feature/<template>/"`
	FeatureName  string            `json:"feature_name"       jsonschema:"required,description:Feature package name"`
	Template     string            `json:"template,omitempty" jsonschema:"description:Feature template: service|job|gateway"`
	Force        bool              `json:"force,omitempty"    jsonschema:"description:Overwrite an existing non-empty feature directory"`
	Custom       map[string]string `json:"custom,omitempty"   jsonschema:"description:Custom key/value replacements for templates"`
}

type createFeatureOutput struct {
	WorkspaceDir string `json:"workspace_dir"`
	FeatureDir   string `json:"feature_dir"`
	Template     string `json:"template"`
	Stdout       string `json:"stdout,omitempty"`
	Stderr       string `json:"stderr,omitempty"`
}

// ---------------------------------------------------------------------------
// Tool registrars
// ---------------------------------------------------------------------------

func registerCreateWorkspace(srv *mcpsdk.Server, cfg serverConfig) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        toolCreateWorkspace,
		Description: cfg.kb.ToolDescription(toolCreateWorkspace),
	}, func(
		ctx context.Context, _ *mcpsdk.CallToolRequest, in createWorkspaceInput,
	) (*mcpsdk.CallToolResult, createWorkspaceOutput, error) {
		args := buildWorkspaceArgs(in)

		stdout, stderr, err := cfg.cmdRunner.Run(ctx, "", cfg.executable, args...)
		if err != nil {
			return nil, createWorkspaceOutput{}, fmt.Errorf(
				errWorkspaceSetupFailed,
				err,
				strings.TrimSpace(stderr),
			)
		}

		out := createWorkspaceOutput{
			RepoDir: in.RepoDir,
			Stdout:  stdout,
			Stderr:  stderr,
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: msgWorkspaceCreated},
			},
		}, out, nil
	})
}

func registerCreateFeature(srv *mcpsdk.Server, cfg serverConfig) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        toolCreateFeature,
		Description: cfg.kb.ToolDescription(toolCreateFeature),
	}, func(
		ctx context.Context, _ *mcpsdk.CallToolRequest, in createFeatureInput,
	) (*mcpsdk.CallToolResult, createFeatureOutput, error) {
		templateName := in.Template
		if strings.TrimSpace(templateName) == "" {
			templateName = serviceTemplateName
		}

		args := buildFeatureArgs(in, templateName)

		stdout, stderr, err := cfg.cmdRunner.Run(ctx, in.WorkspaceDir, cfg.executable, args...)
		if err != nil {
			return nil, createFeatureOutput{}, fmt.Errorf(
				errFeatureSetupFailed,
				err,
				strings.TrimSpace(stderr),
			)
		}

		out := createFeatureOutput{
			WorkspaceDir: in.WorkspaceDir,
			FeatureDir:   in.FeatureDir,
			Template:     templateName,
			Stdout:       stdout,
			Stderr:       stderr,
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{Text: msgFeatureCreated},
			},
		}, out, nil
	})
}

// ---------------------------------------------------------------------------
// Arg builders (shared with plan_service and implement_service)
// ---------------------------------------------------------------------------

func buildWorkspaceArgs(in createWorkspaceInput) []string {
	args := []string{
		"setup",
		"workspace",
		"--repoDir", in.RepoDir,
		"--repoModule", in.RepoModule,
	}
	if in.Force {
		args = append(args, "--force")
	}

	args = append(args, flattenCustomArgs(in.Custom)...)

	return args
}

func buildFeatureArgs(in createFeatureInput, templateName string) []string {
	args := []string{
		"setup",
		"feature",
		"--repoModule", in.RepoModule,
		"--featureDir", in.FeatureDir,
		"--featureName", in.FeatureName,
		"--template", templateName,
	}
	if in.Force {
		args = append(args, "--force")
	}

	args = append(args, flattenCustomArgs(in.Custom)...)

	return args
}

func flattenCustomArgs(custom map[string]string) []string {
	if len(custom) == 0 {
		return nil
	}

	keys := make([]string, 0, len(custom))
	for key := range custom {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	args := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		args = append(args, "--custom", fmt.Sprintf("%s=%s", key, custom[key]))
	}

	return args
}
