package langchaingo

import (
	"context"
	"sync"

	"github.com/clubpay/ronykit/intent"
	lclang "github.com/tmc/langchaingo/llms"
)

// Model adapts langchaingo llms.Model to intent.LLM.
type Model struct {
	model lclang.Model
	info  intent.Model
}

// NewModel wraps a langchaingo model.
func NewModel(model lclang.Model, info intent.Model) *Model {
	return &Model{model: model, info: info}
}

func (m *Model) Model() intent.Model {
	if m.info.ID == "" && m.info.Name == "" {
		return intent.Model{Name: "langchaingo"}
	}

	return m.info
}

func (m *Model) Generate(ctx context.Context, req intent.Request) (intent.Response, error) {
	msgs, err := toMessageContent(req.Messages)
	if err != nil {
		return intent.Response{}, err
	}

	opts := toCallOptions(req)

	resp, err := m.model.GenerateContent(ctx, msgs, opts...)
	if err != nil {
		return intent.Response{}, err
	}

	return fromContentResponse(resp), nil
}

func (m *Model) Stream(ctx context.Context, req intent.Request) (intent.Stream, error) {
	ch := make(chan intent.Chunk, 8)
	stream := &channelStream{ch: ch}

	stream.wg.Go(func() {
		defer close(ch)

		msgs, err := toMessageContent(req.Messages)
		if err != nil {
			stream.setErr(err)

			return
		}

		opts := toCallOptions(req)
		opts = append(opts, lclang.WithStreamingFunc(func(streamCtx context.Context, chunk []byte) error {
			select {
			case <-streamCtx.Done():
				return streamCtx.Err()
			case ch <- intent.Chunk{Content: string(chunk)}:
				return nil
			}
		}))

		resp, err := m.model.GenerateContent(ctx, msgs, opts...)
		if err != nil {
			stream.setErr(err)

			return
		}

		final := fromContentResponse(resp)
		ch <- intent.Chunk{
			Content:          final.Content,
			ReasoningContent: final.ReasoningContent,
			ToolCalls:        final.ToolCalls,
			FinishReason:     final.FinishReason,
			Done:             true,
		}
	})

	return stream, nil
}

type channelStream struct {
	ch    <-chan intent.Chunk
	wg    sync.WaitGroup
	err   error
	errMu sync.Mutex
}

func (s *channelStream) setErr(err error) {
	s.errMu.Lock()
	s.err = err
	s.errMu.Unlock()
}

func (s *channelStream) Recv(ctx context.Context) (intent.Chunk, error) {
	select {
	case <-ctx.Done():
		return intent.Chunk{}, ctx.Err()
	case chunk, ok := <-s.ch:
		if ok {
			return chunk, nil
		}
	}

	s.wg.Wait()
	s.errMu.Lock()
	defer s.errMu.Unlock()

	if s.err != nil {
		return intent.Chunk{}, s.err
	}

	return intent.Chunk{Done: true}, nil
}

func (s *channelStream) Close() error { return nil }

var _ intent.LLM = (*Model)(nil)

func toMessageContent(messages []intent.Message) ([]lclang.MessageContent, error) {
	out := make([]lclang.MessageContent, 0, len(messages))
	for _, msg := range messages {
		role := toRole(msg.Role)

		parts := make([]lclang.ContentPart, 0, len(msg.Parts))
		for _, part := range msg.Parts {
			parts = append(parts, lclang.TextContent{Text: part.Text})
		}

		out = append(out, lclang.MessageContent{Role: role, Parts: parts})
	}

	return out, nil
}

func toRole(role intent.Role) lclang.ChatMessageType {
	switch role {
	case intent.RoleSystem:
		return lclang.ChatMessageTypeSystem
	case intent.RoleHuman:
		return lclang.ChatMessageTypeHuman
	case intent.RoleAI:
		return lclang.ChatMessageTypeAI
	case intent.RoleTool:
		return lclang.ChatMessageTypeTool
	case intent.RoleGeneric:
		return lclang.ChatMessageTypeGeneric
	default:
		return lclang.ChatMessageTypeHuman
	}
}

func toCallOptions(req intent.Request) []lclang.CallOption {
	var opts []lclang.CallOption
	if req.Options.Model != "" {
		opts = append(opts, lclang.WithModel(req.Options.Model))
	}

	if req.Options.Temperature != nil {
		opts = append(opts, lclang.WithTemperature(*req.Options.Temperature))
	}

	if req.Options.MaxTokens != nil {
		opts = append(opts, lclang.WithMaxTokens(*req.Options.MaxTokens))
	}

	if len(req.Options.StopWords) > 0 {
		opts = append(opts, lclang.WithStopWords(req.Options.StopWords))
	}

	if req.Options.JSONMode {
		opts = append(opts, lclang.WithJSONMode())
	}

	if req.Options.CandidateCount > 0 {
		opts = append(opts, lclang.WithCandidateCount(req.Options.CandidateCount))
	}

	if len(req.Tools) > 0 {
		opts = append(opts, lclang.WithTools(toFunctionDefinitions(req.Tools)))
	}

	return opts
}

func toFunctionDefinitions(tools []intent.ToolDefinition) []lclang.Tool {
	out := make([]lclang.Tool, 0, len(tools))
	for _, tool := range tools {
		out = append(out, lclang.Tool{
			Type: "function",
			Function: &lclang.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
				Strict:      tool.Strict,
			},
		})
	}

	return out
}

func fromContentResponse(resp *lclang.ContentResponse) intent.Response {
	if resp == nil || len(resp.Choices) == 0 {
		return intent.Response{}
	}

	choices := make([]intent.Choice, 0, len(resp.Choices))
	for _, choice := range resp.Choices {
		choices = append(choices, intent.Choice{
			Content:          choice.Content,
			ToolCalls:        fromToolCalls(choice.ToolCalls),
			FinishReason:     choice.StopReason,
			ReasoningContent: choice.ReasoningContent,
			GenerationInfo:   choice.GenerationInfo,
		})
	}

	first := choices[0]

	return intent.Response{
		Content:          first.Content,
		ToolCalls:        first.ToolCalls,
		FinishReason:     first.FinishReason,
		ReasoningContent: first.ReasoningContent,
		GenerationInfo:   first.GenerationInfo,
		Choices:          choices,
	}
}

func fromToolCalls(calls []lclang.ToolCall) []intent.ToolCall {
	out := make([]intent.ToolCall, 0, len(calls))
	for _, call := range calls {
		item := intent.ToolCall{ID: call.ID, Type: call.Type}
		if call.FunctionCall != nil {
			item.Name = call.FunctionCall.Name
			item.Arguments = call.FunctionCall.Arguments
		}

		out = append(out, item)
	}

	return out
}
