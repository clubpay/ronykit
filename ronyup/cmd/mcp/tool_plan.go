package mcp

import (
	"context"
	"fmt"
	"path"

	"github.com/clubpay/ronykit/ronyup/knowledge"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

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
	RecommendedPackages     []toolkitPackage  `json:"recommended_packages"`
	PlannedFiles            []servicePlanFile `json:"planned_files"`
	NextSteps               []string          `json:"next_steps"`
}

// ---------------------------------------------------------------------------
// Tool registrar
// ---------------------------------------------------------------------------

func registerPlanService(srv *mcpsdk.Server, cfg serverConfig) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        toolPlanService,
		Description: cfg.kb.ToolDescription(toolPlanService),
	}, func(
		_ context.Context, _ *mcpsdk.CallToolRequest, in planServiceInput,
	) (*mcpsdk.CallToolResult, planServiceOutput, error) {
		featureDir, err := normalizeRelativePath(in.FeatureDir)
		if err != nil {
			return nil, planServiceOutput{}, fmt.Errorf(errInvalidFeatureDir, err)
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

		out := planServiceOutput{
			WorkspaceDir:            inspection.AbsDir,
			FeatureDir:              featureDir,
			FeatureName:             featureName,
			Template:                templateName,
			SetupCommand:            append([]string{"ronyup"}, setupArgs...),
			ExistingServiceFeatures: inspection.ExistingServiceFeatures,
			ArchitectureHints:       cfg.kb.ArchitectureHintTexts(),
			CharacteristicHints:     cfg.kb.CharacteristicHints(in.Characteristics),
			RecommendedPackages:     packagesFromKB(cfg.kb),
			PlannedFiles:            buildServicePlanFiles(cfg.kb, featurePath, in.Characteristics),
			NextSteps:               cfg.kb.PlanNextSteps,
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{
					Text: fmt.Sprintf(msgPlannedFiles, len(out.PlannedFiles), featurePath),
				},
			},
		}, out, nil
	})
}

// ---------------------------------------------------------------------------
// Plan-building helpers
// ---------------------------------------------------------------------------

func buildServicePlanFiles(kb *knowledge.Base, featurePath string, characteristics []string) []servicePlanFile {
	purposes := kb.FilePurposes

	spec := []struct {
		templatePath string
		destPath     string
		purposeKey   string
	}{
		{
			templatePath: "skeleton/feature/service/service.gotmpl",
			destPath:     path.Join(featurePath, "service.go"),
			purposeKey:   "service_lifecycle",
		},
		{
			templatePath: "skeleton/feature/service/module.gotmpl",
			destPath:     path.Join(featurePath, "module.go"),
			purposeKey:   "module_wiring",
		},
		{
			templatePath: "skeleton/feature/service/api/service.gotmpl",
			destPath:     path.Join(featurePath, "api", "service.go"),
			purposeKey:   "api_contracts",
		},
		{
			templatePath: "skeleton/feature/service/internal/app/app.gotmpl",
			destPath:     path.Join(featurePath, "internal", "app", "app.go"),
			purposeKey:   "app_usecases",
		},
		{
			templatePath: "skeleton/feature/service/internal/repo/port.go",
			destPath:     path.Join(featurePath, "internal", "repo", "port.go"),
			purposeKey:   "repo_ports",
		},
		{
			templatePath: "skeleton/feature/service/internal/repo/v0/adapter.gotmpl",
			destPath:     path.Join(featurePath, "internal", "repo", "v0", "adapter.go"),
			purposeKey:   "repo_adapters",
		},
		{
			templatePath: "skeleton/feature/service/migration.gotmpl",
			destPath:     path.Join(featurePath, "migration.go"),
			purposeKey:   "migration",
		},
		{
			templatePath: "skeleton/feature/service/internal/settings/settings.gotmpl",
			destPath:     path.Join(featurePath, "internal", "settings", "settings.go"),
			purposeKey:   "settings",
		},
	}

	out := make([]servicePlanFile, 0, len(spec))
	for _, s := range spec {
		out = append(out, servicePlanFile{
			Path:         s.destPath,
			TemplatePath: s.templatePath,
			Purpose:      purposes[s.purposeKey],
			Hints:        kb.FileHints(s.destPath, characteristics),
		})
	}

	return out
}

func packagesFromKB(kb *knowledge.Base) []toolkitPackage {
	out := make([]toolkitPackage, 0, len(kb.Packages))
	for _, p := range kb.Packages {
		out = append(out, toolkitPackage{
			ImportPath:  p.ImportPath,
			ShortName:   p.ShortName,
			Description: p.Description,
			UsageHint:   p.UsageHint,
		})
	}

	return out
}
