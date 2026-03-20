package knowledge

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load parses the embedded knowledge content and returns a Base.
func Load() (*Base, error) {
	return LoadFS(content)
}

// LoadFS parses knowledge from the given filesystem (useful for testing).
func LoadFS(fsys fs.FS) (*Base, error) {
	var base Base

	if err := loadServerInstructions(fsys, &base); err != nil {
		return nil, fmt.Errorf("server instructions: %w", err)
	}

	if err := loadTools(fsys, &base); err != nil {
		return nil, fmt.Errorf("tools: %w", err)
	}

	if err := loadPackages(fsys, &base); err != nil {
		return nil, fmt.Errorf("packages: %w", err)
	}

	if err := loadArchitecture(fsys, &base); err != nil {
		return nil, fmt.Errorf("architecture: %w", err)
	}

	if err := loadCharacteristics(fsys, &base); err != nil {
		return nil, fmt.Errorf("characteristics: %w", err)
	}

	if err := loadPlanNextSteps(fsys, &base); err != nil {
		return nil, fmt.Errorf("plan next steps: %w", err)
	}

	if err := loadFilePurposes(fsys, &base); err != nil {
		return nil, fmt.Errorf("file purposes: %w", err)
	}

	if err := loadPrompts(fsys, &base); err != nil {
		return nil, fmt.Errorf("prompts: %w", err)
	}

	return &base, nil
}

func loadServerInstructions(fsys fs.FS, base *Base) error {
	data, err := fs.ReadFile(fsys, "server/instructions.md")
	if err != nil {
		return err
	}

	base.ServerInstructions = strings.TrimSpace(string(data))

	return nil
}

func loadTools(fsys fs.FS, base *Base) error {
	base.Tools = make(map[string]ToolDoc)

	entries, err := fs.ReadDir(fsys, "tools")
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		data, err := fs.ReadFile(fsys, "tools/"+e.Name())
		if err != nil {
			return err
		}

		var doc ToolDoc

		body, err := parseFrontmatter(data, &doc)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}

		sections := splitSections(body)
		doc.Description = strings.TrimSpace(sections[""])
		doc.ExtendedGuidance = strings.TrimSpace(sections["Extended Guidance"])

		if doc.Name == "" {
			doc.Name = strings.TrimSuffix(e.Name(), ".md")
		}

		base.Tools[doc.Name] = doc
	}

	return nil
}

func loadPackages(fsys fs.FS, base *Base) error {
	entries, err := fs.ReadDir(fsys, "packages")
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		data, err := fs.ReadFile(fsys, "packages/"+e.Name())
		if err != nil {
			return err
		}

		var doc PackageDoc

		body, err := parseFrontmatter(data, &doc)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}

		sections := splitSections(body)
		doc.Description = strings.TrimSpace(sections[""])
		doc.UsageHint = strings.TrimSpace(sections["Usage Hint"])

		if doc.ShortName == "" {
			doc.ShortName = strings.TrimSuffix(e.Name(), ".md")
		}

		base.Packages = append(base.Packages, doc)
	}

	return nil
}

func loadArchitecture(fsys fs.FS, base *Base) error {
	entries, err := fs.ReadDir(fsys, "architecture")
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		data, err := fs.ReadFile(fsys, "architecture/"+e.Name())
		if err != nil {
			return err
		}

		base.ArchitectureHints = append(base.ArchitectureHints, ArchitectureHint{
			Slug: strings.TrimSuffix(e.Name(), ".md"),
			Text: strings.TrimSpace(string(data)),
		})
	}

	return nil
}

func loadCharacteristics(fsys fs.FS, base *Base) error {
	entries, err := fs.ReadDir(fsys, "characteristics")
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		data, err := fs.ReadFile(fsys, "characteristics/"+e.Name())
		if err != nil {
			return err
		}

		var doc CharacteristicDoc

		body, err := parseFrontmatter(data, &doc)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}

		sections := splitSections(body)
		doc.ServiceHint = strings.TrimSpace(sections[""])
		doc.FileHint = strings.TrimSpace(sections["File-Level Hint"])

		base.Characteristics = append(base.Characteristics, doc)
	}

	return nil
}

func loadPlanNextSteps(fsys fs.FS, base *Base) error {
	data, err := fs.ReadFile(fsys, "planning/next-steps.md")
	if err != nil {
		return err
	}

	base.PlanNextSteps = parseOrderedList(string(data))

	return nil
}

func loadFilePurposes(fsys fs.FS, base *Base) error {
	data, err := fs.ReadFile(fsys, "planning/file-purposes.md")
	if err != nil {
		return err
	}

	var purposes map[string]string
	if _, err := parseFrontmatter(data, &purposes); err != nil {
		return err
	}

	base.FilePurposes = purposes

	return nil
}

func loadPrompts(fsys fs.FS, base *Base) error {
	entries, err := fs.ReadDir(fsys, "prompts")
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		data, err := fs.ReadFile(fsys, "prompts/"+e.Name())
		if err != nil {
			return err
		}

		var doc PromptDoc

		body, err := parseFrontmatter(data, &doc)
		if err != nil {
			return fmt.Errorf("%s: %w", e.Name(), err)
		}

		doc.Template = strings.TrimSpace(body)

		if doc.Name == "" {
			doc.Name = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		}

		base.Prompts = append(base.Prompts, doc)
	}

	return nil
}

// parseFrontmatter splits YAML frontmatter from body. If the content does
// not start with "---\n", dest is left untouched and the full content is
// returned as the body.
func parseFrontmatter(data []byte, dest any) (body string, err error) {
	const sep = "---"

	s := string(data)
	if !strings.HasPrefix(s, sep+"\n") {
		return s, nil
	}

	rest := s[len(sep)+1:]

	before, after, ok := strings.Cut(rest, "\n"+sep+"\n")
	if !ok {
		// Try end-of-file closing fence.
		if strings.HasSuffix(strings.TrimRight(rest, "\n"), sep) {
			trimmed := strings.TrimRight(rest, "\n")

			yamlBlock := trimmed[:len(trimmed)-len(sep)]
			if err := yaml.Unmarshal([]byte(yamlBlock), dest); err != nil {
				return "", fmt.Errorf("parse frontmatter yaml: %w", err)
			}

			return "", nil
		}

		return s, nil
	}

	yamlBlock := before
	body = after

	if err := yaml.Unmarshal([]byte(yamlBlock), dest); err != nil {
		return "", fmt.Errorf("parse frontmatter yaml: %w", err)
	}

	return body, nil
}

// splitSections splits markdown body by ## headings. The text before the
// first heading is stored under the empty-string key.
func splitSections(body string) map[string]string {
	sections := map[string]string{}
	currentKey := ""

	var buf strings.Builder

	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, "## ") {
			sections[currentKey] = buf.String()
			currentKey = strings.TrimPrefix(line, "## ")

			buf.Reset()

			continue
		}

		buf.WriteString(line)
		buf.WriteByte('\n')
	}

	sections[currentKey] = buf.String()

	return sections
}

var orderedListRe = regexp.MustCompile(`^\d+\.\s+`)

// parseOrderedList extracts items from a Markdown ordered list.
func parseOrderedList(s string) []string {
	var items []string

	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(line)
		if orderedListRe.MatchString(line) {
			item := orderedListRe.ReplaceAllString(line, "")
			if item != "" {
				items = append(items, item)
			}
		}
	}

	return items
}
