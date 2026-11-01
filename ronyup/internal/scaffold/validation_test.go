package scaffold

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateFeatureName(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"billing", "my_feature", "feature2"} {
		if err := ValidateFeatureName(name); err != nil {
			t.Errorf("ValidateFeatureName(%q): %v", name, err)
		}
	}

	for _, name := range []string{"", "..", "../billing", "billing/admin", `billing\\admin`, "billing-api", "func"} {
		if err := ValidateFeatureName(name); err == nil {
			t.Errorf("ValidateFeatureName(%q) should fail", name)
		}
	}
}

func TestSetupFeatureRejectsTraversalBeforeWriting(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	root := filepath.Join(parent, "workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	err := SetupFeature(context.Background(), FeatureRequest{
		StartDir:    root,
		FeatureDir:  "../outside",
		FeatureName: "outside",
	}, DiscardLogger{})
	if err == nil {
		t.Fatal("expected traversal to be rejected")
	}
	if _, err := os.Lstat(filepath.Join(parent, "outside")); !os.IsNotExist(err) {
		t.Fatalf("feature escaped workspace: %v", err)
	}
}

func TestLoadBundlesConfigRejectsUnsafeBundleName(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	manifest := []byte("bundles:\n  ../outside:\n    services: [\"*\"]\n")
	if err := os.WriteFile(filepath.Join(root, BundlesManifestName), manifest, 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := LoadBundlesConfig(root); err == nil {
		t.Fatal("expected unsafe bundle name to be rejected")
	}
}

func TestPrepareWorkspaceDirRefusesFilesystemRoot(t *testing.T) {
	t.Parallel()

	root := filepath.VolumeName(string(filepath.Separator)) + string(filepath.Separator)
	if err := prepareWorkspaceDir(root, true); err == nil {
		t.Fatal("expected forced root replacement to be rejected")
	}
}

func TestNormalizeFeatureRequest(t *testing.T) {
	t.Parallel()

	req := FeatureRequest{FeatureDir: "feature/billing"}
	if err := normalizeFeatureRequest(&req); err != nil {
		t.Fatal(err)
	}
	if req.FeatureDir != "billing" || req.FeaturePrefix != "feature" || req.FeatureName != "billing" {
		t.Fatalf("unexpected normalized request: %+v", req)
	}
}
