package mcp

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type fakeRunner struct {
	calls    []runnerCall
	runHook  func(cwd, name string, args []string) error
	lastCwd  string
	lastName string
	lastArgs []string
}

type runnerCall struct {
	cwd  string
	name string
	args []string
}

func (f *fakeRunner) Run(_ context.Context, cwd, name string, args ...string) (string, string, error) {
	if f.runHook != nil {
		if err := f.runHook(cwd, name, args); err != nil {
			return "", err.Error(), err
		}
	}
	f.calls = append(f.calls, runnerCall{
		cwd:  cwd,
		name: name,
		args: append([]string{}, args...),
	})
	f.lastCwd = cwd
	f.lastName = name
	f.lastArgs = append([]string{}, args...)

	return "ok", "", nil
}

func TestBuildAPIServiceContent(t *testing.T) {
	content := buildAPIServiceContent(apiServiceContentInput{
		repoModule:      "github.com/example/repo",
		packagePath:     "feature/service/billing",
		featureDir:      "billing",
		featureName:     "billing",
		serviceSummary:  "Billing execution endpoint.",
		characteristics: []string{"postgres", "idempotent"},
		contracts: []normalizedServiceContract{
			{Name: "ExecuteBilling", HTTPMethod: "POST", RoutePath: "/billing/execute"},
			{Name: "ExecuteRefund", HTTPMethod: "POST", RoutePath: "/billing/refund"},
		},
	})

	if !containsAll(content,
		`package api`,
		`rony.POST("/billing/execute"`,
		`rony.POST("/billing/refund"`,
		`svc.billingHandlers()`,
		`svc.refundHandlers()`,
		`func (svc Service) billingHandlers() rony.SetupOption[rony.EMPTY, rony.NOP]`,
		`func (svc Service) refundHandlers() rony.SetupOption[rony.EMPTY, rony.NOP]`,
		`type ExecuteBillingInput struct`,
		`type ExecuteRefundInput struct`,
		`Operation string`,
		`svc.ExecuteBilling`,
		`svc.billingApp.ExecuteBilling(`,
		`errs.B().Cause(err).Msg("EXECUTE_BILLING_FAILED").Err()`,
	) {
		t.Fatalf("generated API content is missing expected fragments:\n%s", content)
	}
}

func TestBuildAppServiceContent(t *testing.T) {
	content := buildAppServiceContent(appServiceContentInput{
		serviceSummary:  "Billing app behavior.",
		characteristics: []string{"redis"},
		contracts: []normalizedServiceContract{
			{Name: "ExecuteBilling"},
		},
	})

	if !containsAll(content,
		`package app`,
		`func (app *App) BuildMessage(operation, payload string) string`,
		`func (app *App) ExecuteBilling(_ context.Context, in ExecuteBillingInput) (*ExecuteBillingOutput, error)`,
		`[]string{"redis"}`,
		`strings.Join(defaultCharacteristics, ",")`,
	) {
		t.Fatalf("generated app content is missing expected fragments:\n%s", content)
	}
}

func TestEnsureFeatureExists_CreateIfMissing(t *testing.T) {
	tmpDir := t.TempDir()
	featurePath := filepath.Join(tmpDir, "feature", "service", "billing")
	runner := &fakeRunner{}

	created, _, _, err := ensureFeatureExists(context.Background(), serverConfig{
		executable: "ronyup",
		cmdRunner:  runner,
	}, ensureFeatureExistsInput{
		workspaceAbsDir: tmpDir,
		featureAbsPath:  featurePath,
		featureDir:      "billing",
		featureName:     "billing",
		repoModule:      "github.com/example/repo",
		createIfMissing: true,
	})
	if err != nil {
		t.Fatalf("ensureFeatureExists failed: %v", err)
	}

	if !created {
		t.Fatalf("expected feature to be created")
	}

	wantArgs := []string{
		"setup", "feature",
		"--repoModule", "github.com/example/repo",
		"--featureDir", "billing",
		"--featureName", "billing",
		"--template", "service",
	}
	if !slices.Equal(runner.lastArgs, wantArgs) {
		t.Fatalf("unexpected args: got=%v want=%v", runner.lastArgs, wantArgs)
	}
}

func TestNormalizeContracts(t *testing.T) {
	contracts, err := normalizeContracts("billing", []serviceContractInput{
		{Name: "billing", HTTPMethod: "post"},
		{Name: "refund", RoutePath: "billing/refund"},
	})
	if err != nil {
		t.Fatalf("normalizeContracts failed: %v", err)
	}

	if len(contracts) != 2 {
		t.Fatalf("expected 2 contracts, got %d", len(contracts))
	}
	if contracts[0].Name != "ExecuteBilling" || contracts[0].RoutePath != "/billing/billing" {
		t.Fatalf("unexpected first contract: %#v", contracts[0])
	}
	if contracts[1].Name != "ExecuteRefund" || contracts[1].RoutePath != "/billing/refund" {
		t.Fatalf("unexpected second contract: %#v", contracts[1])
	}
}

func TestImplementService_EndToEnd(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.work"), []byte("go 1.25.1\n"), 0o644); err != nil {
		t.Fatalf("write go.work: %v", err)
	}

	runner := &fakeRunner{
		runHook: func(cwd, _ string, args []string) error {
			if len(args) >= 2 && args[0] == "setup" && args[1] == "feature" {
				featureDir := ""
				for i := 0; i < len(args)-1; i++ {
					if args[i] == "--featureDir" {
						featureDir = args[i+1]
						break
					}
				}
				if featureDir == "" {
					return nil
				}
				base := filepath.Join(cwd, "feature", "service", featureDir)
				return os.MkdirAll(filepath.Join(base, "internal", "app"), 0o755)
			}

			return nil
		},
	}
	runGoFmt := false
	out, err := implementService(context.Background(), serverConfig{
		executable: "ronyup",
		cmdRunner:  runner,
	}, implementServiceInput{
		WorkspaceDir:    tmpDir,
		RepoModule:      "github.com/example/repo",
		FeatureDir:      "billing",
		FeatureName:     "billing",
		ServiceSummary:  "Billing service",
		Characteristics: []string{"postgres", "redis"},
		Contracts: []serviceContractInput{
			{Name: "create_invoice", HTTPMethod: "POST"},
			{Name: "refund", HTTPMethod: "POST", RoutePath: "/billing/refund"},
		},
		CreateIfMissing: true,
		RunGoFmt:        &runGoFmt,
	})
	if err != nil {
		t.Fatalf("implementService failed: %v", err)
	}

	if !out.CreatedFeature {
		t.Fatalf("expected CreatedFeature=true")
	}
	if len(out.ChangedFiles) != 3 {
		t.Fatalf("expected 3 changed files, got %d", len(out.ChangedFiles))
	}
	if !slices.ContainsFunc(runner.calls, func(c runnerCall) bool {
		return c.name == "ronyup" && len(c.args) >= 2 && c.args[0] == "setup" && c.args[1] == "feature"
	}) {
		t.Fatalf("expected setup feature command to be called")
	}

	apiFile := filepath.Join(tmpDir, "feature", "service", "billing", "api", "service.go")
	appFile := filepath.Join(tmpDir, "feature", "service", "billing", "internal", "app", "app.go")
	repoPortFile := filepath.Join(tmpDir, "feature", "service", "billing", "internal", "repo", "port.go")
	apiContent, err := os.ReadFile(apiFile)
	if err != nil {
		t.Fatalf("read api file: %v", err)
	}
	appContent, err := os.ReadFile(appFile)
	if err != nil {
		t.Fatalf("read app file: %v", err)
	}

	repoPortContent, err := os.ReadFile(repoPortFile)
	if err != nil {
		t.Fatalf("read repo port file: %v", err)
	}

	if !containsAll(string(apiContent), "ExecuteCreateInvoice", "ExecuteRefund", `"/billing/refund"`) {
		t.Fatalf("api file does not include generated contracts:\n%s", string(apiContent))
	}
	if !containsAll(string(apiContent), "svc.invoiceHandlers()", "svc.refundHandlers()") {
		t.Fatalf("api file does not include grouped handler composition:\n%s", string(apiContent))
	}
	if !containsAll(string(appContent), "BuildMessage", "ExecuteCreateInvoice", "ExecuteRefund", "postgres", "redis") {
		t.Fatalf("app file does not include generated app message helpers:\n%s", string(appContent))
	}
	if !containsAll(string(repoPortContent), "type ServiceRepository interface", "ExecuteCreateInvoice", "ExecuteRefund") {
		t.Fatalf("repo port file does not include generated repository contracts:\n%s", string(repoPortContent))
	}
}

func TestNormalizeContracts_GroupPassthrough(t *testing.T) {
	contracts, err := normalizeContracts("billing", []serviceContractInput{
		{Name: "deposit_fiat", Group: "deposit"},
	})
	if err != nil {
		t.Fatalf("normalizeContracts failed: %v", err)
	}

	if len(contracts) != 1 {
		t.Fatalf("expected 1 contract, got %d", len(contracts))
	}
	if contracts[0].Group != "deposit" {
		t.Fatalf("expected group to be preserved, got %q", contracts[0].Group)
	}
}

func TestInferGroupName_PrefersDomainOverVerb(t *testing.T) {
	got := inferGroupName("/billing/createInvoice", "ExecuteCreateInvoice")
	if got != "Invoice" {
		t.Fatalf("unexpected group name: got=%q want=%q", got, "Invoice")
	}
}

func TestInferGroupName_FromRouteHierarchy(t *testing.T) {
	got := inferGroupName("/payment/deposit/fiat/initiate", "ExecuteInitiateFiatDeposit")
	if got != "Deposit" {
		t.Fatalf("unexpected group name: got=%q want=%q", got, "Deposit")
	}
}

func TestInferGroupName_FallbackToContract(t *testing.T) {
	got := inferGroupName("/billing/execute", "ExecuteBilling")
	if got != "Billing" {
		t.Fatalf("unexpected group name: got=%q want=%q", got, "Billing")
	}
}

func TestGroupContracts_UsesExplicitGroup(t *testing.T) {
	groups := groupContracts([]normalizedServiceContract{
		{
			Name:      "ExecuteCreateInvoice",
			RoutePath: "/billing/createInvoice",
			Group:     "payment",
		},
		{
			Name:      "ExecuteRefund",
			RoutePath: "/billing/refund",
		},
	})

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Name != "payment" {
		t.Fatalf("unexpected explicit group: got=%q", groups[0].Name)
	}
}

func TestWriteGeneratedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	files := []generatedFile{
		{RelativePath: "feature/service/billing/api/service.go", Content: "package api\n"},
		{RelativePath: "feature/service/billing/internal/app/app.go", Content: "package app\n"},
	}

	changed, err := writeGeneratedFiles(tmpDir, files)
	if err != nil {
		t.Fatalf("writeGeneratedFiles failed: %v", err)
	}

	want := []string{
		"feature/service/billing/api/service.go",
		"feature/service/billing/internal/app/app.go",
	}
	if !slices.Equal(changed, want) {
		t.Fatalf("unexpected changed files: got=%v want=%v", changed, want)
	}

	for _, rel := range want {
		abs := filepath.Join(tmpDir, filepath.FromSlash(rel))
		if _, err := os.Stat(abs); err != nil {
			t.Fatalf("expected file to exist %s: %v", rel, err)
		}
	}
}

func containsAll(content string, fragments ...string) bool {
	for _, f := range fragments {
		if !strings.Contains(content, f) {
			return false
		}
	}

	return true
}
