package langchaingo_test

import (
	"context"
	"testing"

	lclang "github.com/tmc/langchaingo/llms"

	"github.com/clubpay/ronykit/intent"
	lcadapter "github.com/clubpay/ronykit/std/llms/langchaingo"
)

type fakeLangModel struct {
	lastMessages []lclang.MessageContent
	lastOpts     []lclang.CallOption
}

func (f *fakeLangModel) GenerateContent(_ context.Context, messages []lclang.MessageContent, options ...lclang.CallOption) (*lclang.ContentResponse, error) {
	f.lastMessages = messages
	f.lastOpts = options

	return &lclang.ContentResponse{
		Choices: []*lclang.ContentChoice{{
			Content:    "hello",
			StopReason: "stop",
			ToolCalls: []lclang.ToolCall{{
				ID:   "1",
				Type: "function",
				FunctionCall: &lclang.FunctionCall{
					Name:      "echo",
					Arguments: `{"x":1}`,
				},
			}},
		}},
	}, nil
}

func (f *fakeLangModel) Call(context.Context, string, ...lclang.CallOption) (string, error) {
	return "", nil
}

func TestAdaptGenerate(t *testing.T) {
	fake := &fakeLangModel{}
	model := lcadapter.NewModel(fake, intent.Model{ID: "fake", Name: "Fake"})

	resp, err := model.Generate(context.Background(), intent.Request{
		Messages: []intent.Message{{
			Role:  intent.RoleHuman,
			Parts: []intent.Part{intent.TextPart("hi")},
		}},
		Tools: []intent.ToolDefinition{{
			Name:        "echo",
			Description: "echo",
			Parameters:  map[string]any{"type": "object"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hello" {
		t.Fatalf("got content %q", resp.Content)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "echo" {
		t.Fatalf("unexpected tool calls: %#v", resp.ToolCalls)
	}
	if len(fake.lastMessages) != 1 {
		t.Fatalf("expected one message sent to langchaingo")
	}
}

func TestAdaptStream(t *testing.T) {
	fake := &fakeLangModel{}
	model := lcadapter.NewModel(fake, intent.Model{ID: "fake"})

	stream, err := model.Stream(context.Background(), intent.Request{
		Messages: []intent.Message{{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("hi")}}},
	})
	if err != nil {
		t.Fatal(err)
	}

	chunk, err := stream.Recv(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if chunk.Content != "hello" || !chunk.Done {
		t.Fatalf("unexpected final chunk: %#v", chunk)
	}
}
