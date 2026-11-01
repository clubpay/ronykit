package scaffold

import (
	"fmt"
	"go/token"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/module"
)

func validateRelativePath(label, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", label)
	}

	if value != strings.TrimSpace(value) {
		return fmt.Errorf("%s must not have leading or trailing whitespace", label)
	}

	if strings.ContainsRune(value, '\x00') || strings.Contains(value, `\`) {
		return fmt.Errorf("%s %q contains an invalid path character", label, value)
	}

	if path.IsAbs(value) || filepath.IsAbs(value) || path.Clean(value) != value || value == "." {
		return fmt.Errorf("%s %q must be a clean, workspace-relative path", label, value)
	}

	for part := range strings.SplitSeq(value, "/") {
		if part == ".." {
			return fmt.Errorf("%s %q must not contain parent-directory segments", label, value)
		}
	}

	return nil
}

// ValidatePathSegment verifies that value is one safe directory name.
func ValidatePathSegment(label, value string) error {
	if err := validateRelativePath(label, value); err != nil {
		return err
	}

	if strings.Contains(value, "/") {
		return fmt.Errorf("%s %q must be a single directory name", label, value)
	}

	return nil
}

// ValidateFeatureName verifies the MCP feature name used in paths and Go code.
func ValidateFeatureName(name string) error {
	if err := ValidatePathSegment("feature name", name); err != nil {
		return err
	}

	if !token.IsIdentifier(name) {
		return fmt.Errorf("feature name %q must be a valid Go package identifier", name)
	}

	return nil
}

func validateModulePath(modulePath string) error {
	if err := module.CheckPath(modulePath); err != nil {
		return fmt.Errorf("invalid Go module path %q: %w", modulePath, err)
	}

	return nil
}

func normalizeFeatureRequest(req *FeatureRequest) error {
	req.FeaturePrefix = strings.TrimSpace(req.FeaturePrefix)
	if req.FeaturePrefix == "" {
		req.FeaturePrefix = DefaultFeaturePrefix
	}

	if prefix := req.FeaturePrefix + "/"; strings.HasPrefix(req.FeatureDir, prefix) {
		req.FeatureDir = strings.TrimPrefix(req.FeatureDir, prefix)
	}

	if err := validateRelativePath("featurePrefix", req.FeaturePrefix); err != nil {
		return err
	}

	if err := validateRelativePath("featureDir", req.FeatureDir); err != nil {
		return err
	}

	if req.FeatureName == "" {
		req.FeatureName = path.Base(req.FeatureDir)
	}

	if !token.IsIdentifier(req.FeatureName) {
		return fmt.Errorf("featureName %q must be a valid Go package identifier", req.FeatureName)
	}

	return nil
}
