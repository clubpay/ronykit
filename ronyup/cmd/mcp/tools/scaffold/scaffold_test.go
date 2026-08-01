package scaffold

import (
	"slices"
	"testing"
)

func TestWorkspaceCLIArgs_IncludesRepoDirDot(t *testing.T) {
	t.Parallel()

	got := workspaceCLIArgs(workspaceArgs{
		Path:   "/tmp/ws",
		Kind:   "fullstack",
		Skills: []string{"all"},
	})
	want := []string{
		"setup", "workspace", "--repoDir", ".",
		"--kind", "fullstack",
		"--skills", "all",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("workspaceCLIArgs: got %v want %v", got, want)
	}
}

func TestWorkspaceCLIArgs_Defaults(t *testing.T) {
	t.Parallel()

	got := workspaceCLIArgs(workspaceArgs{Path: "/tmp/ws"})
	want := []string{"setup", "workspace", "--repoDir", "."}
	if !slices.Equal(got, want) {
		t.Fatalf("workspaceCLIArgs: got %v want %v", got, want)
	}
}

func TestFeatureCLIArgs(t *testing.T) {
	t.Parallel()

	got := featureCLIArgs(featureArgs{
		Name:            "billing",
		Template:        "service",
		FeaturePrefix:   "feature",
		GroupByTemplate: true,
	})
	want := []string{
		"setup", "feature",
		"--featureDir", "billing",
		"--featureName", "billing",
		"--template", "service",
		"--featurePrefix", "feature",
		"--groupByTemplate",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("featureCLIArgs: got %v want %v", got, want)
	}
}
