package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/ronyup/internal/z"
	"github.com/clubpay/ronykit/x/rkit"
)

const (
	frameworkSkillID     = "ronykit-framework"
	skillIssueUnsafePath = "unsafe_path"
	skillIssueReadError  = "read_error"
)

// SkillSyncRequest configures the skill-only workspace sync.
type SkillSyncRequest struct {
	StartDir   string
	Kind       string
	SkillsMode []string
	Overwrite  bool
}

// SkillIssue describes one piece of drift from the embedded skill catalog.
type SkillIssue struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

// SkillStatus reports validation issues for one installed skill.
type SkillStatus struct {
	ID      string       `json:"id"`
	Current bool         `json:"current"`
	Issues  []SkillIssue `json:"issues"`
}

// SkillsReport is the result of a read-only skill validation.
type SkillsReport struct {
	WorkspacePath string        `json:"workspacePath"`
	UpToDate      bool          `json:"upToDate"`
	Skills        []SkillStatus `json:"skills"`
}

// SyncSkills refreshes only the framework and selected catalog skills.
func SyncSkills(req SkillSyncRequest, log Logger) error {
	if log == nil {
		log = DiscardLogger{}
	}

	layout, err := ResolveWorkspaceLayout(req.StartDir, req.Kind)
	if err != nil {
		return err
	}

	skillIDs, err := resolveSyncSkills(layout.RepoRoot, req.SkillsMode, layout.Kind)
	if err != nil {
		return err
	}

	input := TemplateInput{
		Kind:   layout.Kind,
		Skills: selectedSkillInfos(skillIDs),
	}
	callback := func(filePath string, _ bool) {
		log.Println("sync:", filePath)
	}

	if err := syncSection(
		SyncSectionSkills,
		layout,
		input,
		skillIDs,
		!req.Overwrite,
		callback,
		log,
	); err != nil {
		return fmt.Errorf("sync skills: %w", err)
	}

	log.Println("Skill sync complete")

	return nil
}

// ValidateSkills compares installed skill files and agent links with the
// embedded catalog without modifying the workspace.
func ValidateSkills(req SkillSyncRequest) (SkillsReport, error) {
	layout, err := ResolveWorkspaceLayout(req.StartDir, req.Kind)
	if err != nil {
		return SkillsReport{}, err
	}

	skillIDs, err := resolveSyncSkills(layout.RepoRoot, req.SkillsMode, layout.Kind)
	if err != nil {
		return SkillsReport{}, err
	}

	report := SkillsReport{
		WorkspacePath: layout.RepoRoot,
		UpToDate:      true,
		Skills:        make([]SkillStatus, 0, len(skillIDs)+1),
	}
	seen := map[string]bool{}

	selected := append([]string{frameworkSkillID}, skillIDs...)
	for _, id := range selected {
		if seen[id] {
			continue
		}

		seen[id] = true
		status := SkillStatus{ID: id, Current: true, Issues: []SkillIssue{}}
		status.Issues = validateInstalledSkill(layout.RepoRoot, id)
		status.Current = len(status.Issues) == 0
		report.Skills = append(report.Skills, status)
	}

	for _, id := range retiredSkillIDs() {
		if seen[id] || !hasInstalledSkill(layout.RepoRoot, id) {
			continue
		}

		seen[id] = true
		replacements := strings.Join(retiredSkills[id], ", ")
		report.Skills = append(report.Skills, SkillStatus{
			ID:      id,
			Current: false,
			Issues: []SkillIssue{
				{
					Kind: "retired_skill",
					Path: filepath.ToSlash(filepath.Join(".agents", "skills", id)),
					Message: fmt.Sprintf(
						"retired skill; sync with overwrite to replace it with %s",
						replacements,
					),
				},
			},
		})
	}

	for _, status := range report.Skills {
		if !status.Current {
			report.UpToDate = false

			break
		}
	}

	return report, nil
}

func retiredSkillIDs() []string {
	ids := rkit.MapKeysToArray(retiredSkills)

	slices.Sort(ids)

	return ids
}

func hasInstalledSkill(repoRoot, id string) bool {
	for _, root := range skillInstallRoots(repoRoot) {
		if _, err := os.Lstat(filepath.Join(root, id)); err == nil {
			return true
		}
	}

	return false
}

func validateInstalledSkill(repoRoot, id string) []SkillIssue {
	issues := []SkillIssue{}
	root := filepath.Join(agentsSkillRoot(repoRoot), id)
	rootRel := relativeSkillPath(repoRoot, root)

	if err := z.CheckPath(repoRoot, root); err != nil {
		return append(issues, SkillIssue{
			Kind:    skillIssueUnsafePath,
			Path:    rootRel,
			Message: err.Error(),
		})
	}

	info, err := os.Lstat(root)
	switch {
	case os.IsNotExist(err):
		issues = append(issues, SkillIssue{
			Kind:    "missing_skill",
			Path:    rootRel,
			Message: "skill is not installed",
		})
	case err != nil:
		issues = append(issues, SkillIssue{
			Kind:    skillIssueReadError,
			Path:    rootRel,
			Message: err.Error(),
		})
	case !info.IsDir():
		issues = append(issues, SkillIssue{
			Kind:    "invalid_skill_root",
			Path:    rootRel,
			Message: "skill root must be a real directory",
		})
	default:
		fileIssues, err := validateSkillFiles(repoRoot, root, id)
		if err != nil {
			issues = append(issues, SkillIssue{
				Kind:    "validation_error",
				Path:    rootRel,
				Message: err.Error(),
			})
		} else {
			issues = append(issues, fileIssues...)
		}
	}

	return append(issues, validateSkillAgentLinks(repoRoot, id)...)
}

//nolint:nilerr // Missing or unsafe files are validation findings; other errors still propagate.
func validateSkillFiles(repoRoot, root, id string) ([]SkillIssue, error) {
	issues := []SkillIssue{}
	srcRoot := skillSourceRoot(id)

	err := fs.WalkDir(
		internal.Skeleton,
		srcRoot,
		func(srcPath string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if entry.IsDir() {
				return nil
			}

			rel := strings.TrimPrefix(strings.TrimPrefix(srcPath, srcRoot), "/")
			dest := filepath.Join(root, filepath.FromSlash(rel))
			destRel := relativeSkillPath(repoRoot, dest)

			if err := z.CheckPath(repoRoot, dest); err != nil {
				issues = append(
					issues,
					SkillIssue{Kind: skillIssueUnsafePath, Path: destRel, Message: err.Error()},
				)

				return nil
			}

			expected, err := internal.Skeleton.ReadFile(srcPath)
			if err != nil {
				return err
			}

			info, err := os.Lstat(dest)
			if os.IsNotExist(err) {
				issues = append(
					issues,
					SkillIssue{Kind: "missing_file", Path: destRel, Message: "file is missing"},
				)

				return nil
			}

			if err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				issues = append(
					issues,
					SkillIssue{Kind: "invalid_file", Path: destRel, Message: "expected a regular file"},
				)

				return nil
			}

			actual, err := os.ReadFile(dest)
			if err != nil {
				return err
			}

			if !bytes.Equal(actual, expected) {
				issues = append(issues, SkillIssue{
					Kind:    "modified_file",
					Path:    destRel,
					Message: "file differs from the embedded skill",
				})
			}

			if rel == "SKILL.md" {
				name, description, ok := parseSkillFrontmatter(string(actual))
				if !ok || name != id || strings.TrimSpace(description) == "" {
					issues = append(issues, SkillIssue{
						Kind:    "invalid_frontmatter",
						Path:    destRel,
						Message: "SKILL.md frontmatter must contain the matching name and a description",
					})
				}
			}

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("validate %s: %w", id, err)
	}

	return issues, nil
}

func validateSkillAgentLinks(repoRoot, id string) []SkillIssue {
	issues := []SkillIssue{}
	link := filepath.Join(repoRoot, ".claude", "skills", id)
	linkRel := relativeSkillPath(repoRoot, link)

	if err := z.CheckPath(repoRoot, filepath.Dir(link)); err != nil {
		issues = append(issues, SkillIssue{Kind: skillIssueUnsafePath, Path: linkRel, Message: err.Error()})
	} else {
		info, err := os.Lstat(link)
		switch {
		case os.IsNotExist(err):
			issues = append(
				issues,
				SkillIssue{Kind: "missing_link", Path: linkRel, Message: "Claude skill link is missing"},
			)
		case err != nil:
			issues = append(
				issues,
				SkillIssue{Kind: skillIssueReadError, Path: linkRel, Message: err.Error()},
			)
		case info.Mode()&os.ModeSymlink == 0:
			issues = append(
				issues,
				SkillIssue{
					Kind:    "invalid_link",
					Path:    linkRel,
					Message: "expected a symlink to the canonical skill",
				},
			)
		default:
			target, err := os.Readlink(link)
			if err != nil {
				issues = append(
					issues,
					SkillIssue{Kind: skillIssueReadError, Path: linkRel, Message: err.Error()},
				)
			} else if target != claudeSkillLinkTarget(id) {
				issues = append(issues, SkillIssue{
					Kind: "invalid_link",
					Path: linkRel,
					Message: fmt.Sprintf(
						"link points to %q instead of %q",
						target,
						claudeSkillLinkTarget(id),
					),
				})
			}
		}
	}

	legacy := filepath.Join(repoRoot, ".cursor", "skills", id)
	if err := z.CheckPath(repoRoot, filepath.Dir(legacy)); err != nil {
		issues = append(issues, SkillIssue{
			Kind:    skillIssueUnsafePath,
			Path:    relativeSkillPath(repoRoot, legacy),
			Message: err.Error(),
		})
	} else if _, err := os.Lstat(legacy); err == nil {
		issues = append(issues, SkillIssue{
			Kind:    "legacy_copy",
			Path:    relativeSkillPath(repoRoot, legacy),
			Message: "legacy Cursor copy can drift; sync with overwrite to remove it",
		})
	} else if !os.IsNotExist(err) {
		issues = append(issues, SkillIssue{
			Kind:    skillIssueReadError,
			Path:    relativeSkillPath(repoRoot, legacy),
			Message: err.Error(),
		})
	}

	return issues
}

func skillSourceRoot(id string) string {
	if id == frameworkSkillID {
		return filepath.ToSlash(filepath.Join("skeleton", "workspace", ".agents", "skills", frameworkSkillID))
	}

	return filepath.ToSlash(filepath.Join(SkillsSrcPrefix, id))
}

func relativeSkillPath(repoRoot, path string) string {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}

	return filepath.ToSlash(rel)
}
