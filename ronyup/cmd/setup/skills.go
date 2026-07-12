package setup

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/ronyup/internal"
	"github.com/clubpay/ronykit/ronyup/internal/z"
)

// SkillDef describes a bundled agent skill that can be pre-installed into a new
// workspace under .agents/skills/<ID>. Skills are curated defaults for testing,
// modern code, formatting, and review workflows.
type SkillDef struct {
	// ID is both the skill directory name and the value used by the --skills flag.
	ID string
	// Name is a human-friendly label shown in the interactive selector.
	Name string
	// Description is a one-line summary shown next to the option.
	Description string
	// Category groups skills in the selector (e.g. "Go", "Frontend", "Quality").
	Category string
	// DefaultBackend selects the skill by default for backend-only workspaces.
	DefaultBackend bool
	// DefaultFullstack selects the skill by default for fullstack workspaces.
	DefaultFullstack bool
}

// SkillInfo is the template-facing view of a selected skill (used by AGENTS.md).
type SkillInfo struct {
	ID          string
	Name        string
	Description string
}

// Skill categories used to group entries in the interactive selector.
const (
	catGo       = "Go"
	catQuality  = "Quality"
	catWorkflow = "Workflow"
	catFrontend = "Frontend"
)

// Special tokens accepted by the --skills flag.
const (
	skillTokenNone     = "none"
	skillTokenAll      = "all"
	skillTokenDefault  = "default"
	skillTokenDefaults = "defaults"
)

// skillCatalog is the curated set of skills bundled with ronyup. Backend
// (Go/quality/workflow) skills default on for every workspace; frontend skills
// default on only for fullstack scaffolds but remain selectable anywhere.
//
// When adding a skill, choose its structure deliberately (single self-contained
// SKILL.md vs. a progressive-disclosure rules/ tree) per the heuristic in
// internal/skeleton/skills/README.md — don't just copy a neighbor's shape.
var skillCatalog = []SkillDef{
	{
		ID:               "go-modern",
		Name:             "Modern Go",
		Description:      "Idiomatic Go 1.25+/1.26 (generics, iterators, errors, context)",
		Category:         catGo,
		DefaultBackend:   true,
		DefaultFullstack: true,
	},
	{
		ID:               "go-testing",
		Name:             "Go Testing",
		Description:      "Table-driven tests, mandatory x/testkit repo integration tests, app unit tests, Ginkgo/Gomega",
		Category:         catGo,
		DefaultBackend:   true,
		DefaultFullstack: true,
	},
	{
		ID:               "writing-tests",
		Name:             "Writing Tests",
		Description:      "Language-agnostic TDD discipline and the test pyramid",
		Category:         catQuality,
		DefaultBackend:   true,
		DefaultFullstack: true,
	},
	{
		ID:               "code-formatting",
		Name:             "Code Formatting & Linting",
		Description:      "Run formatters/linters; keep diffs clean before finishing",
		Category:         catQuality,
		DefaultBackend:   true,
		DefaultFullstack: true,
	},
	{
		ID:               "systematic-debugging",
		Name:             "Systematic Debugging",
		Description:      "Find root cause before fixing; reproduce, verify, prevent",
		Category:         catQuality,
		DefaultBackend:   true,
		DefaultFullstack: true,
	},
	{
		ID:               "code-review",
		Name:             "Code Review",
		Description:      "Prioritized review for correctness, design, and risk",
		Category:         catQuality,
		DefaultBackend:   true,
		DefaultFullstack: true,
	},
	{
		ID:               "verification-before-completion",
		Name:             "Verification Before Completion",
		Description:      "Require fresh evidence before claiming work is done",
		Category:         catQuality,
		DefaultBackend:   true,
		DefaultFullstack: true,
	},
	{
		ID:               "writing-plans",
		Name:             "Writing Plans",
		Description:      "Plan multi-step work into small, testable tasks first",
		Category:         catWorkflow,
		DefaultBackend:   true,
		DefaultFullstack: true,
	},
	{
		ID:               "conventional-commits",
		Name:             "Conventional Commits",
		Description:      "Atomic commits with the Conventional Commits format",
		Category:         catWorkflow,
		DefaultBackend:   true,
		DefaultFullstack: true,
	},
	{
		ID:               "nextjs-modern",
		Name:             "Modern Next.js",
		Description:      "Next.js 16 App Router, Server Components, Server Actions",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
	{
		ID:               "frontend-testing",
		Name:             "Frontend Testing",
		Description:      "Vitest + Testing Library and Playwright e2e best practices",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
	{
		ID:               "storybook",
		Name:             "Component Storybook",
		Description:      "Ensure every UI component ships with a Storybook story",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
	{
		ID:               "shadcn",
		Name:             "shadcn/ui",
		Description:      "Find, install, compose, and theme shadcn/ui components correctly",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
	{
		ID:               "dashboard-ui",
		Name:             "Dashboard & Admin UI",
		Description:      "Lay out dashboards/back-office apps: KPI strip, F-pattern, color discipline",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
	{
		ID:               "frontend-design",
		Name:             "Frontend Design",
		Description:      "Required before UI bootstrap: ask design questions, token plan, avoid generic AI aesthetics",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
	{
		ID:               "ux-quality",
		Name:             "UX Quality",
		Description:      "Accessible, usable web UI: keyboard/focus, interaction states, forms, navigation, motion",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
	{
		ID:               "typography",
		Name:             "Typography",
		Description:      "Correct screen typography: curly quotes, dashes, line length, hierarchy, the JSX entity trap",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
	{
		ID:               "design-tokens",
		Name:             "Design Tokens",
		Description:      "Required with frontend-design: primitive→semantic→component tokens, Tailwind/shadcn wiring",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
	{
		ID:               "react-performance",
		Name:             "React Performance",
		Description:      "70 React/Next.js perf rules: waterfalls, bundles, re-renders (Vercel, MIT)",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
	{
		ID:               "composition-patterns",
		Name:             "React Composition Patterns",
		Description:      "Scalable component APIs: compound components, lifted state (Vercel, MIT)",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
	{
		ID:               "webmcp",
		Name:             "WebMCP",
		Description:      "Expose frontend apps as agent-callable tools via document.modelContext (W3C WebMCP)",
		Category:         catFrontend,
		DefaultBackend:   false,
		DefaultFullstack: true,
	},
}

// skillsSrcPrefix is the embedded source path that holds the bundled skills.
const skillsSrcPrefix = "skeleton/skills"

func skillExists(id string) bool {
	for _, s := range skillCatalog {
		if s.ID == id {
			return true
		}
	}

	return false
}

// defaultSkillIDs returns the skill IDs selected by default for the given
// workspace kind.
func defaultSkillIDs(kind string) []string {
	var ids []string

	for _, s := range skillCatalog {
		var isDefault bool

		switch kind {
		case KindFullstack:
			isDefault = s.DefaultFullstack
		case KindFrontend:
			// Frontend-only workspaces want the frontend defaults plus the
			// language-agnostic quality/workflow skills, but not Go skills.
			isDefault = s.DefaultFullstack && s.Category != catGo
		default:
			isDefault = s.DefaultBackend
		}

		if isDefault {
			ids = append(ids, s.ID)
		}
	}

	return ids
}

// resolveSkillSelection turns the raw --skills values into a validated, ordered
// (catalog order), de-duplicated list of skill IDs. Supported special tokens:
//
//	default  -> the defaults for the workspace kind
//	all      -> every bundled skill
//	none / "" -> no skills
//
// When requested is empty (flag not set), the kind defaults are used.
func resolveSkillSelection(requested []string, kind string) ([]string, error) {
	if len(requested) == 0 {
		return defaultSkillIDs(kind), nil
	}

	wanted := map[string]bool{}

	for _, raw := range requested {
		for part := range strings.SplitSeq(raw, ",") {
			id := strings.TrimSpace(part)
			if id == "" {
				continue
			}

			switch id {
			case skillTokenNone:
				return nil, nil
			case skillTokenDefault, skillTokenDefaults:
				for _, d := range defaultSkillIDs(kind) {
					wanted[d] = true
				}
			case skillTokenAll:
				for _, s := range skillCatalog {
					wanted[s.ID] = true
				}
			default:
				if !skillExists(id) {
					return nil, fmt.Errorf(
						"unknown skill %q (available: %s)",
						id,
						strings.Join(allSkillIDs(), ", "),
					)
				}

				wanted[id] = true
			}
		}
	}

	return filterCatalogOrder(wanted), nil
}

func allSkillIDs() []string {
	ids := make([]string, 0, len(skillCatalog))
	for _, s := range skillCatalog {
		ids = append(ids, s.ID)
	}

	return ids
}

// filterCatalogOrder returns the IDs present in the set, in catalog order.
func filterCatalogOrder(set map[string]bool) []string {
	var ids []string

	for _, s := range skillCatalog {
		if set[s.ID] {
			ids = append(ids, s.ID)
		}
	}

	return ids
}

// selectedSkillInfos returns template-facing metadata for the given IDs, in
// catalog order, so AGENTS.md can list the installed skills.
func selectedSkillInfos(ids []string) []SkillInfo {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}

	var infos []SkillInfo

	for _, s := range skillCatalog {
		if set[s.ID] {
			infos = append(infos, SkillInfo{ID: s.ID, Name: s.Name, Description: s.Description})
		}
	}

	return infos
}

// copySkills copies the selected skill directories from the embedded FS into
// <skillsRoot>/.agents/skills/<id>. skillsRoot is the directory that holds the
// workspace's .agents folder (the repository root for both backend and
// fullstack scaffolds).
func copySkills(skillsRoot string, ids []string, callback func(filePath string, dir bool)) error {
	dest := filepath.Join(skillsRoot, ".agents", "skills")

	for _, id := range ids {
		if !skillExists(id) {
			continue
		}

		err := z.CopyDir(z.CopyDirParams{
			FS:             internal.Skeleton,
			SrcPathPrefix:  filepath.ToSlash(filepath.Join(skillsSrcPrefix, id)),
			DestPathPrefix: filepath.Join(dest, id),
			Callback:       callback,
		})
		if err != nil {
			return fmt.Errorf("copy skill %q: %w", id, err)
		}
	}

	return nil
}
