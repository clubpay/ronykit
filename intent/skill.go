package intent

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/clubpay/ronykit/intent/errs"
	"github.com/clubpay/ronykit/intent/internal/prompt"
)

// ActivateSkillTool is the reserved name of the synthetic tool the runtime
// advertises so the model can load a skill's full instructions on demand.
const ActivateSkillTool = "activate_skill"

// Skill is a named capability the agent can activate on demand.
//
// Only Name and Description are advertised to the model up front (progressive
// disclosure); Instructions and skill-scoped Tools enter the turn only after
// the model activates the skill via the ActivateSkillTool.
type Skill struct {
	Name         string            // unique key; the enum value the model selects
	Description  string            // short summary; always advertised in the catalog
	Instructions string            // full body; injected only on activation
	Tools        []string          // tool names unlocked while the skill is active
	Triggers     []string          // keywords/phrases that suggest this skill (shown in catalog)
	Examples     []string          // example user requests (shown in catalog)
	Meta         map[string]string // optional adapter metadata
}

// Card returns the cheap advertisement view of the skill.
func (s Skill) Card() SkillCard {
	return SkillCard{
		Name:        s.Name,
		Description: s.Description,
		Triggers:    s.Triggers,
		Examples:    s.Examples,
	}
}

// SkillCard is the cheap metadata advertised for a Skill in the catalog.
type SkillCard struct {
	Name        string
	Description string
	Triggers    []string
	Examples    []string
}

// SkillRegistry provides skill discovery (cheap cards) and activation (full body).
type SkillRegistry interface {
	// List returns the advertisement cards for all available skills.
	List(ctx context.Context) ([]SkillCard, error)
	// Get returns the full skill, including Instructions and Tools.
	Get(ctx context.Context, name string) (Skill, error)
}

// DefaultSkillRegistry is an in-memory SkillRegistry.
type DefaultSkillRegistry struct {
	mu     sync.RWMutex
	skills map[string]Skill
}

// NewSkillRegistry returns an empty in-memory skill registry.
func NewSkillRegistry() *DefaultSkillRegistry {
	return &DefaultSkillRegistry{skills: make(map[string]Skill)}
}

// Register adds a skill to the registry. The skill name is required.
func (r *DefaultSkillRegistry) Register(skill Skill) error {
	if r == nil {
		return errs.Wrap(errs.ErrUnsupportedOperation, "skill registry is nil")
	}

	if skill.Name == "" {
		return errs.Wrap(errs.ErrUnsupportedOperation, "skill name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.skills[skill.Name] = skill

	return nil
}

func (r *DefaultSkillRegistry) List(_ context.Context) ([]SkillCard, error) {
	if r == nil {
		return nil, nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]SkillCard, 0, len(r.skills))
	for _, skill := range r.skills {
		out = append(out, skill.Card())
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	return out, nil
}

func (r *DefaultSkillRegistry) Get(_ context.Context, name string) (Skill, error) {
	if r == nil {
		return Skill{}, errs.ErrSkillNotFound
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, ok := r.skills[name]
	if !ok {
		return Skill{}, errs.SkillNotFound(name)
	}

	return skill, nil
}

var _ SkillRegistry = (*DefaultSkillRegistry)(nil)

// SkillCatalogTitle is the heading injected with the per-turn skill catalog.
const SkillCatalogTitle = prompt.SkillCatalogTitle

// FormatSkillCatalog renders the advertised skill routing guide.
func FormatSkillCatalog(cards []SkillCard) string {
	rows := make([]prompt.SkillRow, 0, len(cards))
	for _, c := range cards {
		rows = append(rows, prompt.SkillRow{
			Name:        c.Name,
			Description: c.Description,
			Triggers:    c.Triggers,
			Examples:    c.Examples,
		})
	}

	return prompt.SkillCatalog(prompt.SkillCatalogData{
		Title:             SkillCatalogTitle,
		ActivateSkillTool: ActivateSkillTool,
		Skills:            rows,
	})
}

// userMessageWithCatalog returns a copy of msg with the skill catalog appended.
// The catalog is sent to the LLM only; callers should keep the original message
// for session history.
func userMessageWithCatalog(msg Message, cards []SkillCard) Message {
	if len(cards) == 0 {
		return msg
	}

	catalog := FormatSkillCatalog(cards)

	text := strings.TrimSpace(messagePartsText(msg.Parts))
	if text == "" {
		text = catalog
	} else {
		text = text + "\n\n" + catalog
	}

	return Message{Role: msg.Role, Parts: []Part{TextPart(text)}}
}

func messagePartsText(parts []Part) string {
	var b strings.Builder

	for _, part := range parts {
		if part.Text != "" {
			if b.Len() > 0 {
				b.WriteByte('\n')
			}

			b.WriteString(part.Text)
		}
	}

	return b.String()
}

// activateSkillDefinition builds the synthetic ActivateSkillTool definition whose
// `name` parameter is constrained to the available skill names.
func activateSkillDefinition(cards []SkillCard) ToolDefinition {
	names := make([]string, 0, len(cards))
	for _, c := range cards {
		names = append(names, c.Name)
	}

	catalog := prompt.ActivateSkillToolDescriptionData{CatalogTitle: SkillCatalogTitle}

	return ToolDefinition{
		Name:        ActivateSkillTool,
		Description: prompt.ActivateSkillToolDescription(catalog),
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"enum":        names,
					"description": prompt.ActivateSkillNameDescription(prompt.ActivateSkillNameDescriptionData(catalog)),
				},
			},
			"required": []string{"name"},
		},
	}
}
