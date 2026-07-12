package static

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"

	"gopkg.in/yaml.v3"
)

var kindDirs = map[string]intent.Kind{
	"prompts": intent.KindPrompt,
	"skills":  intent.KindSkill,
	"facts":   intent.KindFact,
}

// Store holds static knowledge loaded from a disk.
type Store struct {
	entries map[string]intent.Entry
	skills  *intent.DefaultSkillRegistry
}

// LoadDir loads prompts, skills, and facts from subdirectories of the root.
// Expected layout: root/{prompts,skills,facts}/<name>.<ext>
//
// Skill files may begin with a YAML front-matter block delimited by `---`
// lines (name, description, tools); the remainder is the skill's instructions.
func LoadDir(root string) (*Store, error) {
	store := &Store{
		entries: make(map[string]intent.Entry),
		skills:  intent.NewSkillRegistry(),
	}

	for dirName, kind := range kindDirs {
		dirPath := filepath.Join(root, dirName)

		info, err := os.Stat(dirPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, err
		}

		if !info.IsDir() {
			return nil, fmt.Errorf("%q is not a directory", dirPath)
		}

		err = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			fileName := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			id := fmt.Sprintf("%s/%s", dirName, fileName)

			if kind == intent.KindSkill {
				return store.loadSkill(id, fileName, path, string(content))
			}

			store.entries[id] = intent.Entry{
				ID:      id,
				Kind:    kind,
				Origin:  intent.OriginStatic,
				Name:    fileName,
				Content: string(content),
				Source:  path,
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return store, nil
}

// skillFrontMatter is the optional YAML header of a skill file.
type skillFrontMatter struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tools       []string `yaml:"tools"`
	Triggers    []string `yaml:"triggers"`
	Examples    []string `yaml:"examples"`
}

func (s *Store) loadSkill(id, fileName, path, content string) error {
	fmText, body, ok := splitFrontMatter(content)

	var fm skillFrontMatter
	if ok {
		if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
			return fmt.Errorf("parse skill front-matter in %q: %w", path, err)
		}
	} else {
		body = content
	}

	name := fm.Name
	if name == "" {
		name = fileName
	}

	instructions := strings.TrimSpace(body)

	err := s.skills.Register(intent.Skill{
		Name:         name,
		Description:  fm.Description,
		Instructions: instructions,
		Tools:        fm.Tools,
		Triggers:     fm.Triggers,
		Examples:     fm.Examples,
		Meta:         map[string]string{"source": path},
	})
	if err != nil {
		return fmt.Errorf("register skill %q: %w", name, err)
	}

	s.entries[id] = intent.Entry{
		ID:      id,
		Kind:    intent.KindSkill,
		Origin:  intent.OriginStatic,
		Name:    name,
		Content: instructions,
		Source:  path,
		Meta:    map[string]string{"description": fm.Description},
	}

	return nil
}

// Skills returns a registry of the loaded skills for use with intent.WithSkills.
func (s *Store) Skills() intent.SkillRegistry {
	if s == nil {
		return nil
	}

	return s.skills
}

// splitFrontMatter separates a leading `---`-delimited YAML block from the body.
// It returns ok=false when the content has no front-matter block.
func splitFrontMatter(content string) (frontMatter, body string, ok bool) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", false
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			fm := strings.Join(lines[1:i], "\n")
			rest := strings.Join(lines[i+1:], "\n")

			return fm, strings.TrimPrefix(rest, "\n"), true
		}
	}

	return "", "", false
}

func (s *Store) List(_ context.Context, filter intent.Filter) ([]intent.Entry, error) {
	if s == nil {
		return nil, nil
	}

	var out []intent.Entry

	for _, entry := range s.entries {
		if !matchesFilter(entry, filter) {
			continue
		}

		out = append(out, entry)
	}

	return out, nil
}

func (s *Store) Get(_ context.Context, id string) (intent.Entry, error) {
	if s == nil {
		return intent.Entry{}, errs.ErrKnowledgeNotFound
	}

	entry, ok := s.entries[id]
	if !ok {
		return intent.Entry{}, errs.ErrKnowledgeNotFound
	}

	return entry, nil
}

func (s *Store) Retrieve(_ context.Context, _ intent.RetrieveQuery) ([]intent.Entry, error) {
	return nil, errs.ErrUnsupportedOperation
}

var (
	_ intent.StaticStore = (*Store)(nil)
	_ intent.Knowledge   = (*Store)(nil)
)

func matchesFilter(entry intent.Entry, filter intent.Filter) bool {
	if len(filter.Kinds) > 0 {
		found := slices.Contains(filter.Kinds, entry.Kind)

		if !found {
			return false
		}
	}

	if len(filter.Names) > 0 {
		found := slices.Contains(filter.Names, entry.Name)

		if !found {
			return false
		}
	}

	return true
}
