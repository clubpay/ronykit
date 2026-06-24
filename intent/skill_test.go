package intent_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/intent"
)

// recordingLLM returns scripted responses and records each request so tests can
// inspect the tool definitions advertised on every iteration.
type recordingLLM struct {
	info      intent.Model
	responses []intent.Response
	requests  []intent.Request
}

func (m *recordingLLM) Model() intent.Model { return m.info }

func (m *recordingLLM) Generate(_ context.Context, req intent.Request) (intent.Response, error) {
	i := len(m.requests)
	m.requests = append(m.requests, req)

	if i >= len(m.responses) {
		return intent.Response{Content: "done"}, nil
	}

	return m.responses[i], nil
}

func (m *recordingLLM) Stream(context.Context, intent.Request) (intent.Stream, error) {
	return nil, errors.New("not implemented")
}

func toolNames(defs []intent.ToolDefinition) []string {
	out := make([]string, 0, len(defs))
	for _, d := range defs {
		out = append(out, d.Name)
	}

	return out
}

func messageText(msgs []intent.Message) string {
	var b strings.Builder
	for _, msg := range msgs {
		for _, part := range msg.Parts {
			b.WriteString(part.Text)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func newSkillAgent(t *testing.T, model intent.LLM, refunded *bool) (*intent.Agent, *intent.Session) {
	t.Helper()

	mem := newRuntimeFakeMemory()
	s, err := intent.NewSessionManager(mem).Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	reg := intent.NewToolRegistry()
	mustRegister(t, reg, "echo", func(_ context.Context, args json.RawMessage) (intent.Message, error) {
		return intent.Message{Role: intent.RoleTool, Parts: []intent.Part{intent.TextPart(string(args))}}, nil
	})
	mustRegister(t, reg, "issue_refund", func(_ context.Context, _ json.RawMessage) (intent.Message, error) {
		*refunded = true

		return intent.Message{Role: intent.RoleTool, Parts: []intent.Part{intent.TextPart("refunded")}}, nil
	})

	skills := intent.NewSkillRegistry()
	err = skills.Register(intent.Skill{
		Name:         "billing",
		Description:  "Handle refunds and billing inquiries.",
		Instructions: "Refund policy: always be kind and verify the order first.",
		Tools:        []string{"issue_refund"},
		Triggers:     []string{"refund", "billing", "charge"},
		Examples:     []string{"refund order 42"},
	})
	if err != nil {
		t.Fatal(err)
	}

	pool, err := intent.NewLLMPool([]intent.LLM{model}, intent.SelectorFunc(intent.SelectFirst))
	if err != nil {
		t.Fatal(err)
	}

	agent := intent.New(
		intent.WithLLMPool(pool),
		intent.WithTools(reg),
		intent.WithSkills(skills),
	)

	return agent, s
}

func mustRegister(t *testing.T, reg *intent.DefaultToolRegistry, name string, fn func(context.Context, json.RawMessage) (intent.Message, error)) {
	t.Helper()

	err := reg.Register(intent.LocalToolFunc{
		Def: intent.ToolDefinition{Name: name},
		Fn:  fn,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunTurnSkillActivation(t *testing.T) {
	var refunded bool

	model := &recordingLLM{
		info: intent.Model{ID: "mock"},
		responses: []intent.Response{
			{ToolCalls: []intent.ToolCall{{ID: "1", Name: intent.ActivateSkillTool, Arguments: `{"name":"billing"}`}}},
			{ToolCalls: []intent.ToolCall{{ID: "2", Name: "issue_refund", Arguments: `{}`}}},
			{Content: "done"},
		},
	}

	agent, s := newSkillAgent(t, model, &refunded)

	result, err := agent.RunTurn(context.Background(), intent.TurnInput{
		Session:     s,
		UserMessage: intent.Message{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("refund order 42")}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Response.Content != "done" {
		t.Fatalf("got %q", result.Response.Content)
	}

	if !refunded {
		t.Fatal("expected issue_refund to run after activation")
	}

	// Iteration 0: skill catalog advertised, gated tool hidden, activate tool present.
	first := toolNames(model.requests[0].Tools)
	if !slices.Contains(first, intent.ActivateSkillTool) {
		t.Fatalf("activate_skill tool not advertised, got %v", first)
	}
	if slices.Contains(first, "issue_refund") {
		t.Fatalf("gated tool should be hidden before activation, got %v", first)
	}
	if !slices.Contains(first, "echo") {
		t.Fatalf("always-on tool missing, got %v", first)
	}
	if !strings.Contains(messageText(model.requests[0].Messages), "Skill Catalog") {
		t.Fatal("skill catalog not injected into context")
	}
	if !strings.Contains(messageText(model.requests[0].Messages), "**billing**") {
		t.Fatal("skill catalog entry missing")
	}
	if !strings.Contains(messageText(model.requests[0].Messages), "Use when: refund, billing, charge") {
		t.Fatal("skill catalog triggers missing")
	}

	for _, tool := range model.requests[0].Tools {
		if tool.Name != intent.ActivateSkillTool {
			continue
		}
		if !strings.Contains(tool.Description, intent.SkillCatalogTitle) {
			t.Fatalf("activate_skill should reference catalog, got %q", tool.Description)
		}
	}

	// Iteration 1: after activation, the skill-scoped tool is unlocked.
	if second := toolNames(model.requests[1].Tools); !slices.Contains(second, "issue_refund") {
		t.Fatalf("gated tool not unlocked after activation, got %v", second)
	}

	// The activation result injected the skill instructions.
	if !strings.Contains(messageText(result.Messages), "Refund policy") {
		t.Fatal("skill instructions not injected on activation")
	}
}

func TestRunTurnGatedToolLocked(t *testing.T) {
	var refunded bool

	model := &recordingLLM{
		info: intent.Model{ID: "mock"},
		responses: []intent.Response{
			{ToolCalls: []intent.ToolCall{{ID: "1", Name: "issue_refund", Arguments: `{}`}}},
			{Content: "done"},
		},
	}

	agent, s := newSkillAgent(t, model, &refunded)

	result, err := agent.RunTurn(context.Background(), intent.TurnInput{
		Session:     s,
		UserMessage: intent.Message{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("refund")}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if refunded {
		t.Fatal("gated tool must not run before its skill is activated")
	}

	if !strings.Contains(messageText(result.Messages), "requires the \"billing\" skill") {
		t.Fatalf("expected a locked message with skill context, got %q", messageText(result.Messages))
	}
}

func TestRunTurnActivateUnknownSkill(t *testing.T) {
	var refunded bool

	model := &recordingLLM{
		info: intent.Model{ID: "mock"},
		responses: []intent.Response{
			{ToolCalls: []intent.ToolCall{{ID: "1", Name: intent.ActivateSkillTool, Arguments: `{"name":"missing"}`}}},
			{Content: "done"},
		},
	}

	agent, s := newSkillAgent(t, model, &refunded)

	result, err := agent.RunTurn(context.Background(), intent.TurnInput{
		Session:     s,
		UserMessage: intent.Message{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("do x")}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(messageText(result.Messages), "missing") {
		t.Fatalf("expected skill-not-found message, got %q", messageText(result.Messages))
	}
}
