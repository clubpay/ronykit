package mcp

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ronyKitModulePath        = "github.com/clubpay/ronykit"
	defaultServiceIdentifier = "Service"
	defaultServiceGroup      = "service"
)

var operationVerbSet = map[string]struct{}{
	"create":   {},
	"get":      {},
	"list":     {},
	"update":   {},
	"delete":   {},
	"remove":   {},
	"set":      {},
	"start":    {},
	"stop":     {},
	"initiate": {},
	"verify":   {},
	"complete": {},
	"cancel":   {},
	"execute":  {},
}

//nolint:lll // Keep explicit jsonschema descriptions for MCP clients.
type implementServiceInput struct {
	WorkspaceDir    string                 `json:"workspace_dir"               jsonschema:"required,description:Workspace root directory that contains go.work"`
	RepoModule      string                 `json:"repo_module"                 jsonschema:"required,description:Go module path for the repository"`
	FeatureDir      string                 `json:"feature_dir"                 jsonschema:"required,description:Feature directory relative to feature/service/"`
	FeatureName     string                 `json:"feature_name"                jsonschema:"required,description:Feature package name in Go identifier format"`
	ServiceSummary  string                 `json:"service_summary,omitempty"   jsonschema:"description:Human-readable service summary used in generated code comments"`
	Characteristics []string               `json:"characteristics,omitempty"   jsonschema:"description:Requested service characteristics like postgres, redis, rest-api, idempotent"`
	Contracts       []serviceContractInput `json:"contracts,omitempty"         jsonschema:"description:Optional contract specs to generate multiple handlers/routes"`
	CreateIfMissing bool                   `json:"create_if_missing,omitempty" jsonschema:"description:Create feature module via setup command when it does not exist"`
	RunGoFmt        *bool                  `json:"run_gofmt,omitempty"         jsonschema:"description:Run gofmt on generated files after writing (default true)"`
}

//nolint:lll // Keep explicit jsonschema descriptions for MCP clients.
type serviceContractInput struct {
	Name       string `json:"name"                  jsonschema:"required,description:Contract operation name in Pascal/camel/snake form"`
	HTTPMethod string `json:"http_method,omitempty" jsonschema:"description:HTTP method for route (default POST)"`
	RoutePath  string `json:"route_path,omitempty"  jsonschema:"description:Absolute route path (default /<feature>/execute)"`
	Group      string `json:"group,omitempty"       jsonschema:"description:Optional handler group name used to compose Desc with svc.<group>Handlers()"`
	Summary    string `json:"summary,omitempty"     jsonschema:"description:Optional summary for docs/comments"`
}

type normalizedServiceContract struct {
	Name       string
	HTTPMethod string
	RoutePath  string
	Group      string
	Summary    string
}

type implementServiceOutput struct {
	WorkspaceDir   string   `json:"workspace_dir"`
	FeatureDir     string   `json:"feature_dir"`
	FeatureName    string   `json:"feature_name"`
	FeaturePath    string   `json:"feature_path"`
	CreatedFeature bool     `json:"created_feature"`
	ChangedFiles   []string `json:"changed_files"`
	Stdout         string   `json:"stdout,omitempty"`
	Stderr         string   `json:"stderr,omitempty"`
}

type generatedFile struct {
	RelativePath string
	Content      string
}

func addImplementServiceTool(srv *mcpsdk.Server, cfg serverConfig) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name: "implement_service",
		Description: "Scaffold and implement service starter code in api/service.go, " +
			"internal/app/app.go, and internal/repo/port.go",
	}, func(
		ctx context.Context, _ *mcpsdk.CallToolRequest, in implementServiceInput,
	) (*mcpsdk.CallToolResult, implementServiceOutput, error) {
		out, err := implementService(ctx, cfg, in)
		if err != nil {
			return nil, implementServiceOutput{}, err
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				&mcpsdk.TextContent{
					Text: fmt.Sprintf("Implemented service starter code in %d file(s).", len(out.ChangedFiles)),
				},
			},
		}, out, nil
	})
}

func implementService(ctx context.Context, cfg serverConfig, in implementServiceInput) (implementServiceOutput, error) {
	featureDir, err := normalizeRelativePath(in.FeatureDir)
	if err != nil {
		return implementServiceOutput{}, fmt.Errorf("invalid feature_dir: %w", err)
	}

	featureName, err := normalizeFeatureName(in.FeatureName)
	if err != nil {
		return implementServiceOutput{}, err
	}

	inspection, err := inspectWorkspace(in.WorkspaceDir)
	if err != nil {
		return implementServiceOutput{}, err
	}

	contracts, err := normalizeContracts(featureDir, in.Contracts)
	if err != nil {
		return implementServiceOutput{}, err
	}

	featurePath := path.Join("feature", "service", featureDir)
	absFeaturePath := filepath.Join(inspection.AbsDir, filepath.FromSlash(featurePath))

	createdFeature, setupStdout, setupStderr, err := ensureFeatureExists(ctx, cfg, ensureFeatureExistsInput{
		workspaceAbsDir: inspection.AbsDir,
		featureAbsPath:  absFeaturePath,
		featureDir:      featureDir,
		featureName:     featureName,
		repoModule:      in.RepoModule,
		createIfMissing: in.CreateIfMissing,
	})
	if err != nil {
		return implementServiceOutput{}, err
	}

	pkgPath := path.Join("feature", "service", featureDir)
	files := []generatedFile{
		{
			RelativePath: path.Join(pkgPath, "api", "service.go"),
			Content: buildAPIServiceContent(apiServiceContentInput{
				repoModule:      in.RepoModule,
				packagePath:     pkgPath,
				featureDir:      featureDir,
				featureName:     featureName,
				serviceSummary:  in.ServiceSummary,
				characteristics: in.Characteristics,
				contracts:       contracts,
			}),
		},
		{
			RelativePath: path.Join(pkgPath, "internal", "app", "app.go"),
			Content: buildAppServiceContent(appServiceContentInput{
				serviceSummary:  in.ServiceSummary,
				characteristics: in.Characteristics,
				contracts:       contracts,
			}),
		},
		{
			RelativePath: path.Join(pkgPath, "internal", "repo", "port.go"),
			Content:      buildRepoPortContent(repoPortContentInput{contracts: contracts}),
		},
	}

	changedFiles, err := writeGeneratedFiles(inspection.AbsDir, files)
	if err != nil {
		return implementServiceOutput{}, err
	}

	runGoFmt := true
	if in.RunGoFmt != nil {
		runGoFmt = *in.RunGoFmt
	}

	if runGoFmt {
		relToFeature := make([]string, 0, len(changedFiles))
		for _, filePath := range changedFiles {
			absFilePath := filepath.Join(inspection.AbsDir, filepath.FromSlash(filePath))

			relPath, relErr := filepath.Rel(absFeaturePath, absFilePath)
			if relErr != nil {
				return implementServiceOutput{}, relErr
			}

			relToFeature = append(relToFeature, relPath)
		}

		sort.Strings(relToFeature)

		_, stderr, err := cfg.cmdRunner.Run(ctx, absFeaturePath, "gofmt", append([]string{"-w"}, relToFeature...)...)
		if err != nil {
			return implementServiceOutput{}, fmt.Errorf("gofmt failed: %w, stderr: %s", err, strings.TrimSpace(stderr))
		}
	}

	return implementServiceOutput{
		WorkspaceDir:   inspection.AbsDir,
		FeatureDir:     featureDir,
		FeatureName:    featureName,
		FeaturePath:    featurePath,
		CreatedFeature: createdFeature,
		ChangedFiles:   changedFiles,
		Stdout:         setupStdout,
		Stderr:         setupStderr,
	}, nil
}

type ensureFeatureExistsInput struct {
	workspaceAbsDir string
	featureAbsPath  string
	featureDir      string
	featureName     string
	repoModule      string
	createIfMissing bool
}

func ensureFeatureExists(
	ctx context.Context, cfg serverConfig, in ensureFeatureExistsInput,
) (bool, string, string, error) {
	_, err := os.Stat(in.featureAbsPath)
	if err == nil {
		return false, "", "", nil
	}

	if !os.IsNotExist(err) {
		return false, "", "", err
	}

	if !in.createIfMissing {
		return false, "", "", fmt.Errorf(
			"feature path does not exist: %s (set create_if_missing=true to scaffold first)",
			in.featureAbsPath,
		)
	}

	args := buildFeatureArgs(createFeatureInput{
		RepoModule:  in.repoModule,
		FeatureDir:  in.featureDir,
		FeatureName: in.featureName,
		Template:    "service",
	}, "service")

	stdout, stderr, err := cfg.cmdRunner.Run(ctx, in.workspaceAbsDir, cfg.executable, args...)
	if err != nil {
		return false, stdout, stderr, fmt.Errorf("create feature failed: %w, stderr: %s", err, strings.TrimSpace(stderr))
	}

	return true, stdout, stderr, nil
}

func writeGeneratedFiles(workspaceAbsDir string, files []generatedFile) ([]string, error) {
	changed := make([]string, 0, len(files))

	for _, file := range files {
		absPath := filepath.Join(workspaceAbsDir, filepath.FromSlash(file.RelativePath))
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return nil, err
		}

		if err := os.WriteFile(absPath, []byte(file.Content), 0o644); err != nil {
			return nil, err
		}

		changed = append(changed, file.RelativePath)
	}

	sort.Strings(changed)

	return changed, nil
}

type apiServiceContentInput struct {
	repoModule      string
	packagePath     string
	featureDir      string
	featureName     string
	serviceSummary  string
	characteristics []string
	contracts       []normalizedServiceContract
}

func buildAPIServiceContent(in apiServiceContentInput) string {
	contracts := in.contracts
	if len(contracts) == 0 {
		contracts = defaultContractsForFeature(in.featureDir)
	}

	fieldName := in.featureName + "App"

	comment := strings.TrimSpace(in.serviceSummary)
	if comment == "" {
		comment = "implements the service operation."
	}

	descEntries := make([]string, 0, len(contracts))
	typeBlocks := make([]string, 0, len(contracts))
	handlerBlocks := make([]string, 0, len(contracts))
	groupedContracts := groupContracts(contracts)
	groupFuncBlocks := make([]string, 0, len(groupedContracts))

	for _, c := range contracts {
		typeBlocks = append(typeBlocks, fmt.Sprintf(`type %sInput struct {
	RequestID string `+"`json:\"request_id,omitempty\"`"+`
	Payload   string `+"`json:\"payload\"`"+`
}

type %sOutput struct {
	Message   string `+"`json:\"message\"`"+`
	Service   string `+"`json:\"service\"`"+`
	Operation string `+"`json:\"operation\"`"+`
}`, c.Name, c.Name))

		handlerComment := c.Summary
		if strings.TrimSpace(handlerComment) == "" {
			handlerComment = c.Name + " validates transport input and delegates to app use-cases."
		}

		handlerBlocks = append(handlerBlocks, fmt.Sprintf(`// %s
func (svc Service) %s(ctx *RContext, input %sInput) (*%sOutput, error) {
	res, err := svc.%s.%s(
		ctx.Context(),
		app.%sInput{
			RequestID: input.RequestID,
			Payload:   input.Payload,
		},
	)
	if err != nil {
		return nil, errs.B().Cause(err).Msg("%s_FAILED").Err()
	}

	return &%sOutput{
		Message:   res.Message,
		Service:   "%s",
		Operation: "%s",
	}, nil
}`,
			handlerComment,
			c.Name,
			c.Name,
			c.Name,
			fieldName,
			c.Name,
			c.Name,
			toScreamingSnake(c.Name),
			c.Name,
			in.featureName,
			c.Name,
		))
	}

	for _, g := range groupedContracts {
		groupFuncName := groupHandlerFuncName(g.Name)
		descEntries = append(descEntries, fmt.Sprintf("\t\tsvc.%s(),", groupFuncName))

		groupEntries := make([]string, 0, len(g.Contracts))
		for _, c := range g.Contracts {
			groupEntries = append(groupEntries, fmt.Sprintf(
				`		rony.WithUnary(
			svc.%s,
			rony.%s("%s", rony.UnaryName("%s")),
		),`,
				c.Name, c.HTTPMethod, c.RoutePath, c.Name,
			))
		}

		groupFuncBlocks = append(groupFuncBlocks, fmt.Sprintf(
			`func (svc Service) %s() rony.SetupOption[rony.EMPTY, rony.NOP] {
	return rony.SetupOptionGroup[rony.EMPTY, rony.NOP](
%s
	)
}`,
			groupFuncName,
			strings.Join(groupEntries, "\n"),
		))
	}

	return fmt.Sprintf(`package api

import (
	"go.uber.org/fx"
	"%s/rony/errs"
	"%s/%s/internal/app"
	"%s/rony"
	"%s/x/settings"
)

// %s
type RContext = rony.UnaryCtx[rony.EMPTY, rony.NOP]

type ServiceParams struct {
	fx.In

	Settings settings.Settings
	App      *app.App
}

type Service struct {
	%s *app.App
}

func New(params ServiceParams) *Service {
	return &Service{
		%s: params.App,
	}
}

func (svc Service) Desc() rony.SetupOption[rony.EMPTY, rony.NOP] {
	return rony.SetupOptionGroup[rony.EMPTY, rony.NOP](
%s
	)
}

%s

%s

%s
`,
		ronyKitModulePath,
		in.repoModule,
		in.packagePath,
		ronyKitModulePath,
		ronyKitModulePath,
		comment,
		fieldName,
		fieldName,
		strings.Join(descEntries, "\n"),
		strings.Join(groupFuncBlocks, "\n\n"),
		strings.Join(typeBlocks, "\n\n"),
		strings.Join(handlerBlocks, "\n\n"),
	)
}

type appServiceContentInput struct {
	serviceSummary  string
	characteristics []string
	contracts       []normalizedServiceContract
}

func buildAppServiceContent(in appServiceContentInput) string {
	comment := strings.TrimSpace(in.serviceSummary)
	if comment == "" {
		comment = "contains application-level service behavior."
	}

	charLiteral := goStringSliceLiteral(in.characteristics)

	useCaseBlocks := make([]string, 0, len(in.contracts))
	for _, c := range in.contracts {
		useCaseBlocks = append(useCaseBlocks, fmt.Sprintf(`type %sInput struct {
	RequestID string
	Payload   string
}

type %sOutput struct {
	Message string
}

func (app *App) %s(_ context.Context, in %sInput) (*%sOutput, error) {
	// TODO: coordinate domain services and repositories for this use-case.
	return &%sOutput{
		Message: app.BuildMessage("%s", in.Payload),
	}, nil
}`,
			c.Name, c.Name, c.Name, c.Name, c.Name, c.Name, c.Name,
		))
	}

	return fmt.Sprintf(`package app

import (
	"context"
	"strings"

	log "%s/x/telemetry/logkit"
	"go.uber.org/fx"
)

// %s
type NewAppParams struct {
	fx.In

	Logger *log.Logger
}

func New(p NewAppParams) (*App, error) {
	app := &App{
		l: p.Logger.With("APP"),
	}

	return app, nil
}

type App struct {
	l *log.Logger
}

var defaultCharacteristics = %s

%s

func (app *App) BuildMessage(operation, payload string) string {
	trimmedPayload := strings.TrimSpace(payload)
	if trimmedPayload == "" {
		trimmedPayload = "empty-payload"
	}

	if len(defaultCharacteristics) == 0 {
		return operation + " processed " + trimmedPayload
	}

	return operation + " processed " + trimmedPayload + " (" + strings.Join(defaultCharacteristics, ",") + ")"
}
`, ronyKitModulePath, comment, charLiteral, strings.Join(useCaseBlocks, "\n\n"))
}

type repoPortContentInput struct {
	contracts []normalizedServiceContract
}

type contractGroup struct {
	Name      string
	Contracts []normalizedServiceContract
}

func buildRepoPortContent(in repoPortContentInput) string {
	methodBlocks := make([]string, 0, len(in.contracts))
	for _, c := range in.contracts {
		methodBlocks = append(methodBlocks, fmt.Sprintf(
			`	// %s persists or fetches data needed by the %s use-case.
	%s(ctx context.Context, requestID, payload string) (string, error)`,
			c.Name, c.Name, c.Name,
		))
	}

	if len(methodBlocks) == 0 {
		methodBlocks = append(methodBlocks, "\t// Add repository methods for your use-cases here.")
	}

	return fmt.Sprintf(`package repo

import "context"

// ServiceRepository defines persistence contracts used by app-level use-cases.
type ServiceRepository interface {
%s
}
`, strings.Join(methodBlocks, "\n\n"))
}

func groupContracts(contracts []normalizedServiceContract) []contractGroup {
	groups := make([]contractGroup, 0, len(contracts))
	index := map[string]int{}

	for _, c := range contracts {
		groupName := strings.TrimSpace(c.Group)
		if groupName == "" {
			groupName = inferGroupName(c.RoutePath, c.Name)
		}

		if groupName == "" {
			groupName = defaultServiceGroup
		}

		i, ok := index[groupName]
		if !ok {
			groups = append(groups, contractGroup{Name: groupName})
			i = len(groups) - 1
			index[groupName] = i
		}

		groups[i].Contracts = append(groups[i].Contracts, c)
	}

	return groups
}

func inferGroupName(routePath, contractName string) string {
	trimmed := strings.Trim(routePath, "/")
	if trimmed != "" {
		parts := strings.Split(trimmed, "/")
		if len(parts) >= 2 && strings.TrimSpace(parts[1]) != "" {
			pathGroup := normalizeGroupCandidate(parts[1])
			if pathGroup != "" {
				return pathGroup
			}
		}
	}

	name := strings.TrimPrefix(contractName, "Execute")

	contractGroup := normalizeGroupCandidate(name)
	if contractGroup != "" {
		return contractGroup
	}

	return defaultServiceGroup
}

func groupHandlerFuncName(group string) string {
	group = sanitizeIdentifierPart(group)
	if group == "" {
		group = defaultServiceGroup
	}

	return lowerCamel(group) + "Handlers"
}

func sanitizeIdentifierPart(v string) string {
	words := splitIdentifierWords(v)

	return joinWordsPascal(words)
}

func splitIdentifierWords(s string) []string {
	if s == "" {
		return nil
	}

	separated := strings.NewReplacer("-", " ", "_", " ", "/", " ").Replace(strings.TrimSpace(s))
	if separated == "" {
		return nil
	}

	baseParts := strings.Fields(separated)
	words := make([]string, 0, len(baseParts))

	for _, p := range baseParts {
		if p == "" {
			continue
		}

		words = append(words, splitCamelToken(p)...)
	}

	return words
}

func splitCamelToken(s string) []string {
	if s == "" {
		return nil
	}

	runes := []rune(s)

	var (
		parts []string
		curr  []rune
	)

	for i, r := range runes {
		isUpper := r >= 'A' && r <= 'Z'
		nextIsLower := i+1 < len(runes) && runes[i+1] >= 'a' && runes[i+1] <= 'z'

		prevIsLower := len(curr) > 0 && curr[len(curr)-1] >= 'a' && curr[len(curr)-1] <= 'z'
		if i > 0 && isUpper && (nextIsLower || prevIsLower) {
			parts = append(parts, strings.ToLower(string(curr)))
			curr = curr[:0]
		}

		curr = append(curr, r)
	}

	if len(curr) > 0 {
		parts = append(parts, strings.ToLower(string(curr)))
	}

	return parts
}

func normalizeGroupCandidate(raw string) string {
	words := splitIdentifierWords(raw)
	if len(words) == 0 {
		return ""
	}

	if _, isVerb := operationVerbSet[words[0]]; isVerb {
		if len(words) == 1 {
			return ""
		}

		words = words[1:]
	}

	return joinWordsPascal(words)
}

func joinWordsPascal(words []string) string {
	if len(words) == 0 {
		return ""
	}

	var out strings.Builder

	for _, w := range words {
		w = strings.TrimSpace(w)
		if w == "" {
			continue
		}

		out.WriteString(toPascalIdentifier(strings.ToLower(w)))
	}

	return out.String()
}

func lowerCamel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}

	runes := []rune(s)
	runes[0] = []rune(strings.ToLower(string(runes[0])))[0]

	return string(runes)
}

func goStringSliceLiteral(values []string) string {
	if len(values) == 0 {
		return "[]string{}"
	}

	clean := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}

		clean = append(clean, fmt.Sprintf("%q", v))
	}

	if len(clean) == 0 {
		return "[]string{}"
	}

	return "[]string{" + strings.Join(clean, ", ") + "}"
}

func toPascalIdentifier(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return defaultServiceIdentifier
	}

	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	})
	if len(parts) == 0 {
		return defaultServiceIdentifier
	}

	var out strings.Builder

	for _, p := range parts {
		if p == "" {
			continue
		}

		out.WriteString(strings.ToUpper(p[:1]))

		if len(p) > 1 {
			out.WriteString(p[1:])
		}
	}

	if out.Len() == 0 {
		return defaultServiceIdentifier
	}

	return out.String()
}

func toScreamingSnake(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "EXECUTE"
	}

	var out strings.Builder

	prev := rune(0)

	for _, r := range s {
		if r == '-' || r == '_' || r == ' ' {
			if out.Len() > 0 && prev != '_' {
				out.WriteRune('_')

				prev = '_'
			}

			continue
		}

		if r >= 'A' && r <= 'Z' && out.Len() > 0 && ((prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9')) {
			out.WriteRune('_')
		}

		orig := r
		if r >= 'a' && r <= 'z' {
			r = r - ('a' - 'A')
		}

		out.WriteRune(r)

		prev = orig
	}

	result := strings.Trim(out.String(), "_")
	if result == "" {
		return "EXECUTE"
	}

	return result
}

func normalizeContracts(featureDir string, in []serviceContractInput) ([]normalizedServiceContract, error) {
	if len(in) == 0 {
		return defaultContractsForFeature(featureDir), nil
	}

	contracts := make([]normalizedServiceContract, 0, len(in))
	seenNames := map[string]struct{}{}

	for _, c := range in {
		name := "Execute" + toPascalIdentifier(c.Name)
		if strings.TrimSpace(c.Name) == "" {
			return nil, fmt.Errorf("contract name is required")
		}

		if _, ok := seenNames[name]; ok {
			return nil, fmt.Errorf("duplicate contract name: %s", name)
		}

		seenNames[name] = struct{}{}

		method := strings.ToUpper(strings.TrimSpace(c.HTTPMethod))
		if method == "" {
			method = "POST"
		}

		switch method {
		case "GET", "POST", "PUT", "PATCH", "DELETE":
		default:
			return nil, fmt.Errorf("unsupported http_method for %s: %s", name, method)
		}

		routePath := strings.TrimSpace(c.RoutePath)
		if routePath == "" {
			routePath = defaultRouteForContract(featureDir, name)
		}

		if !strings.HasPrefix(routePath, "/") {
			routePath = "/" + routePath
		}

		routePath = path.Clean(routePath)
		if !strings.HasPrefix(routePath, "/") {
			routePath = "/" + routePath
		}

		contracts = append(contracts, normalizedServiceContract{
			Name:       name,
			HTTPMethod: method,
			RoutePath:  routePath,
			Group:      strings.TrimSpace(c.Group),
			Summary:    strings.TrimSpace(c.Summary),
		})
	}

	return contracts, nil
}

func defaultContractsForFeature(featureDir string) []normalizedServiceContract {
	name := "Execute" + toPascalIdentifier(path.Base(strings.Trim(featureDir, "/")))

	return []normalizedServiceContract{
		{
			Name:       name,
			HTTPMethod: "POST",
			RoutePath:  defaultRouteForContract(featureDir, name),
			Group:      defaultServiceGroup,
			Summary:    "Default execute contract.",
		},
	}
}

func defaultRouteForContract(featureDir, contractName string) string {
	base := "/" + strings.ReplaceAll(strings.Trim(featureDir, "/"), "_", "-")
	if base == "/" {
		base = "/service"
	}

	suffix := strings.TrimPrefix(contractName, "Execute")
	if suffix == "" {
		return base + "/execute"
	}

	return base + "/" + strings.ToLower(suffix[:1]) + suffix[1:]
}
