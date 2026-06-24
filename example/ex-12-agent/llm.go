package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/clubpay/ronykit/intent"
	lcopenai "github.com/clubpay/ronykit/std/llms/langchaingo"
	ollamaadapter "github.com/clubpay/ronykit/std/llms/ollama"
)

func newLLMPool() (intent.Pool, error) {
	models := []intent.LLM{
		//&mockLLM{info: intent.Model{ID: "mock", Name: "Mock LLM", Priority: 1}},
		ollamaadapter.MustNew(ollamaadapter.WithInfo(intent.Model{ID: "ollama", Priority: 2})),
	}

	if openai, err := lcopenai.NewOpenAI(); err == nil {
		models = append(models, openai)
	} else if strings.TrimSpace(os.Getenv(lcopenai.EnvOpenAIAPIKey)) != "" {
		return nil, err
	}

	return intent.NewLLMPool(models, intent.SelectorFunc(intent.SelectPriority))
}

func modelSummary(pool intent.Pool) string {
	models := pool.Models()
	names := make([]string, 0, len(models))
	for _, model := range models {
		names = append(names, model.Name)
	}

	return strings.Join(names, ", ")
}

type mockLLM struct {
	info intent.Model
}

func (m *mockLLM) Model() intent.Model { return m.info }

func (m *mockLLM) Generate(_ context.Context, req intent.Request) (intent.Response, error) {
	if toolResult, ok := toolResultAfterLastHuman(req.Messages); ok {
		return intent.Response{Content: "The current UTC time is " + toolResult + "."}, nil
	}

	lastHuman := lastHumanMessage(req.Messages)
	lower := strings.ToLower(lastHuman)
	if strings.Contains(lower, "time") && hasTool(req.Tools, "get_time") {
		return intent.Response{
			ToolCalls: []intent.ToolCall{
				{
					ID:        "mock-get-time",
					Type:      "function",
					Name:      "get_time",
					Arguments: `{}`,
				},
			},
		}, nil
	}

	if lastHuman == "" {
		lastHuman = "(empty message)"
	}

	return intent.Response{Content: "Mock agent received: " + lastHuman}, nil
}

func (m *mockLLM) Stream(_ context.Context, req intent.Request) (intent.Stream, error) {
	resp, err := m.Generate(context.Background(), req)
	if err != nil {
		return nil, err
	}

	return &mockStream{resp: resp}, nil
}

type mockStream struct {
	resp   intent.Response
	sent   bool
	closed bool
}

func (s *mockStream) Recv(_ context.Context) (intent.Chunk, error) {
	if s.closed {
		return intent.Chunk{}, fmt.Errorf("stream closed")
	}

	if !s.sent {
		s.sent = true

		return intent.Chunk{
			Content:      s.resp.Content,
			ToolCalls:    s.resp.ToolCalls,
			FinishReason: s.resp.FinishReason,
			Done:         true,
		}, nil
	}

	return intent.Chunk{Done: true}, nil
}

func (s *mockStream) Close() error {
	s.closed = true

	return nil
}

func toolResultAfterLastHuman(messages []intent.Message) (string, bool) {
	lastHumanIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == intent.RoleHuman {
			lastHumanIdx = i

			break
		}
	}

	for i := len(messages) - 1; i > lastHumanIdx; i-- {
		if messages[i].Role != intent.RoleTool || len(messages[i].Parts) == 0 {
			continue
		}

		if messages[i].Parts[0].Text != "" {
			return messages[i].Parts[0].Text, true
		}
	}

	return "", false
}

func lastHumanMessage(messages []intent.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role != intent.RoleHuman {
			continue
		}

		var parts []string
		for _, part := range messages[i].Parts {
			if part.Text != "" {
				parts = append(parts, part.Text)
			}
		}

		return strings.Join(parts, "\n")
	}

	return ""
}

func hasTool(tools []intent.ToolDefinition, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}

	return false
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}

	return fallback
}
