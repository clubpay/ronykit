package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clubpay/ronykit/intent"
)

// Model adapts Ollama's /api/chat endpoint to intent.LLM with tool support.
type Model struct {
	client    *client
	modelName string
	info      intent.Model
}

// New returns an intent.LLM backed by Ollama.
// Unset fields are populated from OLLAMA_MODEL and OLLAMA_BASE_URL when present.
func New(opts ...Option) (*Model, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	baseURL := strings.TrimSpace(cfg.baseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	modelName := strings.TrimSpace(cfg.modelName)
	if modelName == "" {
		return nil, fmt.Errorf("ollama model name is required")
	}

	info := cfg.info
	if info.ID == "" {
		info.ID = "ollama"
	}

	if info.Name == "" {
		info.Name = modelName
	}

	return &Model{
		client:    newClient(baseURL, nil),
		modelName: modelName,
		info:      info,
	}, nil
}

// MustNew is like New but panics on error.
func MustNew(opts ...Option) *Model {
	model, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return model
}

func (m *Model) Model() intent.Model { return m.info }

func (m *Model) Generate(ctx context.Context, req intent.Request) (intent.Response, error) {
	chatReq := chatRequest{
		Model:    m.resolveModel(req),
		Messages: toChatMessages(req.Messages),
		Tools:    toTools(req.Tools),
		Stream:   false,
		Options:  toOptions(req),
	}

	if req.Options.JSONMode {
		chatReq.Format = "json"
	}

	resp, err := m.client.chat(ctx, chatReq)
	if err != nil {
		return intent.Response{}, err
	}

	return fromChatResponse(resp), nil
}

func (m *Model) Stream(ctx context.Context, req intent.Request) (intent.Stream, error) {
	resp, err := m.Generate(ctx, req)
	if err != nil {
		return nil, err
	}

	return &singleChunkStream{resp: resp}, nil
}

var _ intent.LLM = (*Model)(nil)

type singleChunkStream struct {
	resp   intent.Response
	sent   bool
	closed bool
}

func (s *singleChunkStream) Recv(_ context.Context) (intent.Chunk, error) {
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

func (s *singleChunkStream) Close() error {
	s.closed = true

	return nil
}

func (m *Model) resolveModel(req intent.Request) string {
	if req.Options.Model != "" {
		return req.Options.Model
	}

	return m.modelName
}

func toOptions(req intent.Request) map[string]any {
	opts := make(map[string]any)

	if req.Options.Temperature != nil {
		opts["temperature"] = *req.Options.Temperature
	}

	if req.Options.MaxTokens != nil {
		opts["num_predict"] = *req.Options.MaxTokens
	}

	if len(req.Options.StopWords) > 0 {
		opts["stop"] = req.Options.StopWords
	}

	if len(opts) == 0 {
		return nil
	}

	return opts
}

func toTools(tools []intent.ToolDefinition) []tool {
	if len(tools) == 0 {
		return nil
	}

	out := make([]tool, 0, len(tools))
	for _, def := range tools {
		params := def.Parameters
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}

		out = append(out, tool{
			Type: toolTypeFunction,
			Function: toolFunction{
				Name:        def.Name,
				Description: def.Description,
				Parameters:  params,
			},
		})
	}

	return out
}

func toChatMessages(messages []intent.Message) []chatMessage {
	out := make([]chatMessage, 0, len(messages))
	for _, msg := range messages {
		out = append(out, toChatMessage(msg))
	}

	return out
}

func toChatMessage(msg intent.Message) chatMessage {
	role := toRole(msg.Role)
	content := messageText(msg)

	switch msg.Role {
	case intent.RoleAI:
		return chatMessage{
			Role:      role,
			Content:   content,
			ToolCalls: toToolCalls(msg.ToolCalls),
		}
	case intent.RoleTool:
		toolName := msg.ToolName
		if toolName == "" {
			toolName = msg.ToolCallID
		}

		return chatMessage{
			Role:       role,
			Content:    content,
			ToolName:   toolName,
			ToolCallID: msg.ToolCallID,
		}
	default:
		return chatMessage{
			Role:    role,
			Content: content,
		}
	}
}

func toRole(role intent.Role) string {
	switch role {
	case intent.RoleSystem:
		return "system"
	case intent.RoleHuman:
		return "user"
	case intent.RoleAI:
		return "assistant"
	case intent.RoleTool:
		return "tool"
	default:
		return "user"
	}
}

func toToolCalls(calls []intent.ToolCall) []toolCall {
	if len(calls) == 0 {
		return nil
	}

	out := make([]toolCall, 0, len(calls))
	for i, call := range calls {
		id := call.ID
		if id == "" {
			id = fmt.Sprintf("call_%d", i)
		}

		out = append(out, toolCall{
			ID:   id,
			Type: toolTypeFunction,
			Function: toolCallFunction{
				Index:     i,
				Name:      call.Name,
				Arguments: json.RawMessage(normalizeArguments(call.Arguments)),
			},
		})
	}

	return out
}

func normalizeArguments(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}

	return raw
}

func messageText(msg intent.Message) string {
	var parts []string

	for _, part := range msg.Parts {
		if part.Text != "" {
			parts = append(parts, part.Text)
		}
	}

	return strings.Join(parts, "\n")
}

func fromChatResponse(resp chatResponse) intent.Response {
	out := intent.Response{
		Content:      resp.Message.Content,
		ToolCalls:    fromToolCalls(resp.Message.ToolCalls),
		FinishReason: resp.DoneReason,
	}

	if len(out.ToolCalls) > 0 && out.FinishReason == "" {
		out.FinishReason = "tool_calls"
	}

	return out
}

func fromToolCalls(calls []toolCall) []intent.ToolCall {
	if len(calls) == 0 {
		return nil
	}

	out := make([]intent.ToolCall, 0, len(calls))
	for i, call := range calls {
		id := call.ID
		if id == "" {
			id = fmt.Sprintf("call_%d", i)
		}

		out = append(out, intent.ToolCall{
			ID:        id,
			Type:      toolTypeFunction,
			Name:      call.Function.Name,
			Arguments: argumentsToJSON(call.Function.Arguments),
		})
	}

	return out
}

func argumentsToJSON(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return "{}"
	}

	return string(raw)
}
