package mcp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type runner interface {
	Run(ctx context.Context, cwd, name string, args ...string) (stdout, stderr string, err error)
}

type defaultRunner struct{}

func (defaultRunner) Run(ctx context.Context, cwd, name string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}

type templateFile struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

type listTemplatesOutput struct {
	Templates []templateFile `json:"templates"`
}

type readTemplateInput struct {
	Path string `json:"path" jsonschema:"required,description:Template path listed by list_templates"`
}

type readTemplateOutput struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

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

//nolint:lll // Keep explicit jsonschema descriptions for MCP clients.
type planServiceInput struct {
	WorkspaceDir    string   `json:"workspace_dir"             jsonschema:"required,description:Workspace root directory that contains go.work"`
	RepoModule      string   `json:"repo_module"               jsonschema:"required,description:Go module path for the repository"`
	FeatureDir      string   `json:"feature_dir"               jsonschema:"required,description:Feature directory relative to feature/service/"`
	FeatureName     string   `json:"feature_name"              jsonschema:"required,description:Feature package name in Go identifier format"`
	Characteristics []string `json:"characteristics,omitempty" jsonschema:"description:Requested service characteristics like postgres, redis, rest-api, idempotent"`
}

type servicePlanFile struct {
	Path         string   `json:"path"`
	TemplatePath string   `json:"template_path"`
	Purpose      string   `json:"purpose"`
	Hints        []string `json:"hints,omitempty"`
}

type planServiceOutput struct {
	WorkspaceDir            string            `json:"workspace_dir"`
	FeatureDir              string            `json:"feature_dir"`
	FeatureName             string            `json:"feature_name"`
	Template                string            `json:"template"`
	SetupCommand            []string          `json:"setup_command"`
	ExistingServiceFeatures []string          `json:"existing_service_features"`
	ArchitectureHints       []string          `json:"architecture_hints"`
	CharacteristicHints     map[string]string `json:"characteristic_hints"`
	PlannedFiles            []servicePlanFile `json:"planned_files"`
	NextSteps               []string          `json:"next_steps"`
}

type workspaceInspection struct {
	AbsDir                  string
	ExistingServiceFeatures []string
}

var goIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

const serviceTemplateName = "service"

func newServer(cfg serverConfig) *mcpsdk.Server {
	options := &mcpsdk.ServerOptions{
		Instructions: cfg.instructions,
	}
	srv := mcpsdk.NewServer(
		&mcpsdk.Implementation{
			Name:    cfg.name,
			Version: cfg.version,
			Title:   "RonyUP MCP Server",
		},
		options,
	)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "list_templates",
		Description: "List embedded ronyup scaffold files/templates available for project and feature generation",
	}, func(
		_ context.Context, _ *mcpsdk.CallToolRequest, _ struct{},
	) (*mcpsdk.CallToolResult, listTemplatesOutput, error) {
		files, err := collectSkeletonFiles(cfg.skeletonFS)
		if err != nil {
			return nil, listTemplatesOutput{}, err
		}

		out := listTemplatesOutput{
			Templates: files,
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{
					Text: fmt.Sprintf("Found %d scaffold files/templates.", len(files)),
				},
			},
		}, out, nil
	})

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "read_template",
		Description: "Read an embedded ronyup scaffold file/template by path",
	}, func(
		_ context.Context, _ *mcpsdk.CallToolRequest, in readTemplateInput,
	) (*mcpsdk.CallToolResult, readTemplateOutput, error) {
		normalizedPath, err := normalizeTemplatePath(in.Path)
		if err != nil {
			return nil, readTemplateOutput{}, err
		}

		data, err := fs.ReadFile(cfg.skeletonFS, normalizedPath)
		if err != nil {
			return nil, readTemplateOutput{}, err
		}

		out := readTemplateOutput{
			Path:    normalizedPath,
			Content: string(data),
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{
					Text: fmt.Sprintf("Loaded template %s (%d bytes).", normalizedPath, len(data)),
				},
			},
		}, out, nil
	})

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "create_workspace",
		Description: "Create a new RonyKIT workspace by delegating to `ronyup setup workspace`",
	}, func(
		ctx context.Context, _ *mcpsdk.CallToolRequest, in createWorkspaceInput,
	) (*mcpsdk.CallToolResult, createWorkspaceOutput, error) {
		args := buildWorkspaceArgs(in)

		stdout, stderr, err := cfg.cmdRunner.Run(ctx, "", cfg.executable, args...)
		if err != nil {
			return nil, createWorkspaceOutput{}, fmt.Errorf(
				"workspace setup failed: %w, stderr: %s",
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
				&mcpsdk.TextContent{Text: "Workspace created successfully."},
			},
		}, out, nil
	})

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "create_feature",
		Description: "Add a feature module to an existing RonyKIT workspace via `ronyup setup feature`",
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
			return nil, createFeatureOutput{}, fmt.Errorf("feature setup failed: %w, stderr: %s", err, strings.TrimSpace(stderr))
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
				&mcpsdk.TextContent{Text: "Feature created successfully."},
			},
		}, out, nil
	})

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "plan_service",
		Description: "Create a dry-run implementation plan for a new service feature based on requested characteristics",
	}, func(
		_ context.Context, _ *mcpsdk.CallToolRequest, in planServiceInput,
	) (*mcpsdk.CallToolResult, planServiceOutput, error) {
		featureDir, err := normalizeRelativePath(in.FeatureDir)
		if err != nil {
			return nil, planServiceOutput{}, fmt.Errorf("invalid feature_dir: %w", err)
		}

		featureName, err := normalizeFeatureName(in.FeatureName)
		if err != nil {
			return nil, planServiceOutput{}, err
		}

		inspection, err := inspectWorkspace(in.WorkspaceDir)
		if err != nil {
			return nil, planServiceOutput{}, err
		}

		templateName := serviceTemplateName
		featurePath := path.Join("feature", templateName, featureDir)
		setupArgs := buildFeatureArgs(createFeatureInput{
			RepoModule:  in.RepoModule,
			FeatureDir:  featureDir,
			FeatureName: featureName,
			Template:    templateName,
		}, templateName)

		hints := characteristicHints(in.Characteristics)
		out := planServiceOutput{
			WorkspaceDir:            inspection.AbsDir,
			FeatureDir:              featureDir,
			FeatureName:             featureName,
			Template:                templateName,
			SetupCommand:            append([]string{"ronyup"}, setupArgs...),
			ExistingServiceFeatures: inspection.ExistingServiceFeatures,
			ArchitectureHints:       architectureHints(),
			CharacteristicHints:     hints,
			PlannedFiles:            buildServicePlanFiles(featurePath, in.Characteristics),
			NextSteps: []string{
				"Run create_feature (or execute setup_command) to scaffold files using the standard service template.",
				"Implement contracts in api/service.go; keep handlers thin and delegate to app use-cases.",
				"Implement business orchestration in internal/app and keep persistence behind internal/repo/port.go interfaces.",
				"If persistence is needed, add adapter implementations under internal/repo/v0 and update migration/settings files.",
				"Run go test ./... and go fmt ./... in the new feature module before committing.",
			},
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{
					Text: fmt.Sprintf("Planned %d files for service feature %s.", len(out.PlannedFiles), featurePath),
				},
			},
		}, out, nil
	})

	addImplementServiceTool(srv, cfg)

	return srv
}

func collectSkeletonFiles(skeletonFS fs.FS) ([]templateFile, error) {
	var files []templateFile

	err := fs.WalkDir(skeletonFS, "skeleton", func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		files = append(files, templateFile{
			Path: filePath,
			Kind: fileKind(filePath),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

func normalizeTemplatePath(filePath string) (string, error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "", errors.New("template path is required")
	}

	cleanPath := path.Clean(strings.TrimPrefix(filePath, "/"))
	if cleanPath == "." || strings.HasPrefix(cleanPath, "../") || cleanPath == ".." {
		return "", fmt.Errorf("invalid template path: %q", filePath)
	}

	if !strings.HasPrefix(cleanPath, "skeleton/") {
		cleanPath = path.Join("skeleton", cleanPath)
	}

	return cleanPath, nil
}

func fileKind(filePath string) string {
	if strings.HasSuffix(filePath, ".gotmpl") ||
		strings.HasSuffix(filePath, ".yamltmpl") ||
		strings.HasSuffix(filePath, "tmpl") {
		return "template"
	}

	return "file"
}

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

func normalizeRelativePath(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", errors.New("path is required")
	}

	cleanPath := path.Clean(strings.TrimPrefix(v, "/"))
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", fmt.Errorf("path must be relative and without traversal: %q", v)
	}

	return cleanPath, nil
}

func normalizeFeatureName(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", errors.New("feature_name is required")
	}

	if !goIdentifierPattern.MatchString(v) {
		return "", fmt.Errorf("feature_name must be a valid Go identifier: %q", v)
	}

	return v, nil
}

func inspectWorkspace(workspaceDir string) (workspaceInspection, error) {
	absDir, err := filepath.Abs(workspaceDir)
	if err != nil {
		return workspaceInspection{}, err
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return workspaceInspection{}, err
	}

	if !info.IsDir() {
		return workspaceInspection{}, fmt.Errorf("workspace_dir must be a directory: %q", workspaceDir)
	}

	_, err = os.Stat(filepath.Join(absDir, "go.work"))
	if err != nil {
		return workspaceInspection{}, fmt.Errorf("workspace_dir must contain go.work: %w", err)
	}

	featuresRoot := filepath.Join(absDir, "feature", "service")

	entries, err := os.ReadDir(featuresRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return workspaceInspection{AbsDir: absDir}, nil
		}

		return workspaceInspection{}, err
	}

	existing := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		existing = append(existing, entry.Name())
	}

	sort.Strings(existing)

	return workspaceInspection{
		AbsDir:                  absDir,
		ExistingServiceFeatures: existing,
	}, nil
}

func buildServicePlanFiles(featurePath string, characteristics []string) []servicePlanFile {
	spec := []struct {
		templatePath string
		destPath     string
		purpose      string
	}{
		{
			templatePath: "skeleton/feature/service/service.gotmpl",
			destPath:     path.Join(featurePath, "service.go"),
			purpose:      "Service module lifecycle and registration.",
		},
		{
			templatePath: "skeleton/feature/service/module.gotmpl",
			destPath:     path.Join(featurePath, "module.go"),
			purpose:      "Dependency graph wiring for app, api, settings, and infra.",
		},
		{
			templatePath: "skeleton/feature/service/api/service.gotmpl",
			destPath:     path.Join(featurePath, "api", "service.go"),
			purpose:      "Contract routes, input/output DTOs, and thin transport handlers.",
		},
		{
			templatePath: "skeleton/feature/service/internal/app/app.gotmpl",
			destPath:     path.Join(featurePath, "internal", "app", "app.go"),
			purpose:      "Application use-cases and business orchestration logic.",
		},
		{
			templatePath: "skeleton/feature/service/internal/repo/port.go",
			destPath:     path.Join(featurePath, "internal", "repo", "port.go"),
			purpose:      "Repository interfaces consumed by app layer.",
		},
		{
			templatePath: "skeleton/feature/service/internal/repo/v0/adapter.gotmpl",
			destPath:     path.Join(featurePath, "internal", "repo", "v0", "adapter.go"),
			purpose:      "Concrete datasource adapters implementing repository ports.",
		},
		{
			templatePath: "skeleton/feature/service/migration.gotmpl",
			destPath:     path.Join(featurePath, "migration.go"),
			purpose:      "Database migration bundle bootstrap.",
		},
		{
			templatePath: "skeleton/feature/service/internal/settings/settings.gotmpl",
			destPath:     path.Join(featurePath, "internal", "settings", "settings.go"),
			purpose:      "Runtime settings model for DB/Redis and module config.",
		},
	}

	out := make([]servicePlanFile, 0, len(spec))
	for _, s := range spec {
		out = append(out, servicePlanFile{
			Path:         s.destPath,
			TemplatePath: s.templatePath,
			Purpose:      s.purpose,
			Hints:        fileHints(s.destPath, characteristics),
		})
	}

	return out
}

func characteristicHints(characteristics []string) map[string]string {
	hints := map[string]string{}

	for _, raw := range characteristics {
		c := normalizeCharacteristic(raw)
		if c == "" {
			continue
		}

		switch {
		case strings.Contains(c, "postgres"),
			strings.Contains(c, "mysql"),
			strings.Contains(c, "sql"),
			strings.Contains(c, "database"):
			hints[raw] = "Model persistence with repo ports + v0 adapters; update migrations/sqlc and settings."
		case strings.Contains(c, "redis"), strings.Contains(c, "cache"):
			hints[raw] = "Expose cache dependency via settings and keep cache logic in app/repo layers."
		case strings.Contains(c, "rest"), strings.Contains(c, "http"), strings.Contains(c, "api"):
			hints[raw] = "Focus on api/service.go contracts, request validation, and route semantics."
		case strings.Contains(c, "idempotent"):
			hints[raw] = "Design app/repo writes to be retry-safe; handlers should remain deterministic."
		default:
			hints[raw] = "Implement as app-layer use-case behavior and expose through explicit contracts."
		}
	}

	return hints
}

func fileHints(filePath string, characteristics []string) []string {
	filePath = strings.ToLower(filePath)
	seen := map[string]struct{}{}
	hints := make([]string, 0, 4)

	addHint := func(h string) {
		if _, ok := seen[h]; ok {
			return
		}

		seen[h] = struct{}{}
		hints = append(hints, h)
	}

	for _, raw := range characteristics {
		c := normalizeCharacteristic(raw)
		switch {
		case strings.Contains(c, "postgres"),
			strings.Contains(c, "mysql"),
			strings.Contains(c, "sql"),
			strings.Contains(c, "database"):
			if strings.Contains(filePath, "/repo/") ||
				strings.HasSuffix(filePath, "migration.go") ||
				strings.Contains(filePath, "/settings/") {
				addHint("Add persistence contracts in repo ports and implement mappings in v0 adapters.")
			}
		case strings.Contains(c, "redis"), strings.Contains(c, "cache"):
			if strings.Contains(filePath, "/repo/") ||
				strings.Contains(filePath, "/settings/") ||
				strings.Contains(filePath, "/app/") {
				addHint("Add cache behavior with explicit settings and keep logic in app/repo layers.")
			}
		case strings.Contains(c, "rest"), strings.Contains(c, "http"), strings.Contains(c, "api"):
			if strings.Contains(filePath, "/api/") {
				addHint("Define explicit contracts/DTOs and validate request input at API boundary.")
			}
		case strings.Contains(c, "idempotent"):
			if strings.Contains(filePath, "/api/") ||
				strings.Contains(filePath, "/app/") ||
				strings.Contains(filePath, "/repo/") {
				addHint("Ensure behavior is idempotent and safe under retries and duplicate deliveries.")
			}
		}
	}

	return hints
}

func architectureHints() []string {
	return []string{
		"Keep API handlers thin: validate input and delegate business behavior to internal/app.",
		"Use internal/repo/port.go for persistence abstractions consumed by app layer.",
		"Implement concrete data integrations under internal/repo/v0 adapters.",
		"Keep shared cross-module utilities in pkg/* and avoid coupling them to feature business logic.",
	}
}

func normalizeCharacteristic(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	v = strings.ReplaceAll(v, "_", " ")
	v = strings.ReplaceAll(v, "-", " ")

	return v
}
