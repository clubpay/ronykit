package mcp

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/clubpay/ronykit/ronyup/internal"
)

func TestCollectSkeletonFiles(t *testing.T) {
	files, err := collectSkeletonFiles(internal.Skeleton)
	if err != nil {
		t.Fatalf("collectSkeletonFiles failed: %v", err)
	}

	if len(files) == 0 {
		t.Fatalf("expected at least one skeleton file")
	}

	if !slices.ContainsFunc(files, func(v templateFile) bool {
		return v.Path == "skeleton/workspace/Makefile"
	}) {
		t.Fatalf("expected workspace Makefile to be discoverable")
	}

	if !slices.ContainsFunc(files, func(v templateFile) bool {
		return v.Path == "skeleton/feature/service/service.gotmpl" && v.Kind == "template"
	}) {
		t.Fatalf("expected service template to be tagged as template")
	}
}

func TestNormalizeTemplatePath(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantPath  string
		expectErr bool
	}{
		{
			name:     "full path remains stable",
			input:    "skeleton/workspace/Makefile",
			wantPath: "skeleton/workspace/Makefile",
		},
		{
			name:     "relative path gets prefixed",
			input:    "workspace/Makefile",
			wantPath: "skeleton/workspace/Makefile",
		},
		{
			name:      "reject path traversal",
			input:     "../secret.txt",
			expectErr: true,
		},
		{
			name:      "reject empty path",
			input:     "  ",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeTemplatePath(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tc.input)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.wantPath {
				t.Fatalf("unexpected path: got=%q want=%q", got, tc.wantPath)
			}
		})
	}
}

func TestBuildWorkspaceArgs(t *testing.T) {
	args := buildWorkspaceArgs(createWorkspaceInput{
		RepoDir:    "./my-repo",
		RepoModule: "github.com/example/my-repo",
		Force:      true,
		Custom: map[string]string{
			"zeta":  "2",
			"alpha": "1",
		},
	})

	want := []string{
		"setup",
		"workspace",
		"--repoDir", "./my-repo",
		"--repoModule", "github.com/example/my-repo",
		"--force",
		"--custom", "alpha=1",
		"--custom", "zeta=2",
	}

	if !slices.Equal(args, want) {
		t.Fatalf("unexpected args:\n got: %#v\nwant: %#v", args, want)
	}
}

func TestBuildFeatureArgs(t *testing.T) {
	args := buildFeatureArgs(createFeatureInput{
		RepoModule:  "github.com/example/my-repo",
		FeatureDir:  "auth",
		FeatureName: "auth",
		Force:       true,
		Custom: map[string]string{
			"serviceName": "AuthService",
		},
	}, "service")

	want := []string{
		"setup",
		"feature",
		"--repoModule", "github.com/example/my-repo",
		"--featureDir", "auth",
		"--featureName", "auth",
		"--template", "service",
		"--force",
		"--custom", "serviceName=AuthService",
	}

	if !slices.Equal(args, want) {
		t.Fatalf("unexpected args:\n got: %#v\nwant: %#v", args, want)
	}
}

func TestNormalizeRelativePath(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		expectErr bool
	}{
		{name: "simple", input: "billing", want: "billing"},
		{name: "nested", input: "billing/invoice", want: "billing/invoice"},
		{name: "absolute gets normalized", input: "/billing", want: "billing"},
		{name: "traversal rejected", input: "../billing", expectErr: true},
		{name: "empty rejected", input: "   ", expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeRelativePath(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestNormalizeFeatureName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      string
		expectErr bool
	}{
		{name: "valid", input: "billing", want: "billing"},
		{name: "underscore valid", input: "billing_v2", want: "billing_v2"},
		{name: "invalid space", input: "billing service", expectErr: true},
		{name: "invalid hyphen", input: "billing-service", expectErr: true},
		{name: "empty", input: "", expectErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeFeatureName(tc.input)
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got != tc.want {
				t.Fatalf("unexpected value: got=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestInspectWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "go.work"), []byte("go 1.25.1\n"), 0o644); err != nil {
		t.Fatalf("write go.work: %v", err)
	}

	for _, name := range []string{"billing", "auth"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, "feature", "service", name), 0o755); err != nil {
			t.Fatalf("mkdir feature dir: %v", err)
		}
	}

	inspection, err := inspectWorkspace(tmpDir)
	if err != nil {
		t.Fatalf("inspectWorkspace failed: %v", err)
	}

	want := []string{"auth", "billing"}
	if !slices.Equal(inspection.ExistingServiceFeatures, want) {
		t.Fatalf("unexpected features: got=%v want=%v", inspection.ExistingServiceFeatures, want)
	}
}

func TestBuildServicePlanFiles(t *testing.T) {
	files := buildServicePlanFiles("feature/service/billing", []string{"postgres", "rest-api", "idempotent"})
	if len(files) == 0 {
		t.Fatalf("expected planned files")
	}

	if !slices.ContainsFunc(files, func(v servicePlanFile) bool {
		return v.Path == "feature/service/billing/api/service.go" && len(v.Hints) > 0
	}) {
		t.Fatalf("expected api/service.go with characteristic hints")
	}

	if !slices.ContainsFunc(files, func(v servicePlanFile) bool {
		return v.Path == "feature/service/billing/internal/repo/v0/adapter.go" && len(v.Hints) > 0
	}) {
		t.Fatalf("expected repository adapter with DB-oriented hints")
	}
}

func TestNewServer_DoesNotPanicOnSchemaTags(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("newServer panicked: %v", r)
		}
	}()

	_ = newServer(serverConfig{
		name:         "ronyup",
		version:      "v0.0.0-test",
		instructions: "test",
		skeletonFS:   internal.Skeleton,
		cmdRunner:    defaultRunner{},
	})
}
