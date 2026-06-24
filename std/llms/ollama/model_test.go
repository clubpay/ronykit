package ollama_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/intent"
	ollamaadapter "github.com/clubpay/ronykit/std/llms/ollama"
)

func TestMustNew(t *testing.T) {
	model := ollamaadapter.MustNew(
		ollamaadapter.WithModelName("test"),
		ollamaadapter.WithBaseURL("http://127.0.0.1:11434"),
	)
	if model.Model().Name != "test" {
		t.Fatalf("name = %q", model.Model().Name)
	}
}

func TestGenerateSendsTools(t *testing.T) {
	var captured map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			http.NotFound(w, r)

			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		err = json.Unmarshal(body, &captured)
		if err != nil {
			t.Fatal(err)
		}

		_, _ = w.Write([]byte(`{
			"model":"test",
			"message":{"role":"assistant","content":"ok"},
			"done":true,
			"done_reason":"stop"
		}`))
	}))
	defer srv.Close()

	model, err := ollamaadapter.New(
		ollamaadapter.WithModelName("test"),
		ollamaadapter.WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = model.Generate(context.Background(), intent.Request{
		Messages: []intent.Message{{
			Role:  intent.RoleHuman,
			Parts: []intent.Part{intent.TextPart("hello")},
		}},
		Tools: []intent.ToolDefinition{{
			Name:        "get_time",
			Description: "Returns the current UTC time.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	tools, ok := captured["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("expected one tool in request, got %#v", captured["tools"])
	}

	if captured["stream"] != false {
		t.Fatalf("expected non-streaming chat, got %#v", captured["stream"])
	}
}

func TestGenerateParsesToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"model":"test",
			"message":{
				"role":"assistant",
				"content":"",
				"tool_calls":[
					{
						"id":"call_1",
						"function":{"name":"get_time","arguments":{}}
					}
				]
			},
			"done":true,
			"done_reason":"stop"
		}`))
	}))
	defer srv.Close()

	model, err := ollamaadapter.New(
		ollamaadapter.WithModelName("test"),
		ollamaadapter.WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := model.Generate(context.Background(), intent.Request{
		Messages: []intent.Message{{
			Role:  intent.RoleHuman,
			Parts: []intent.Part{intent.TextPart("What time is it?")},
		}},
		Tools: []intent.ToolDefinition{{Name: "get_time"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected one tool call, got %#v", resp.ToolCalls)
	}

	if resp.ToolCalls[0].Name != "get_time" {
		t.Fatalf("unexpected tool call: %#v", resp.ToolCalls[0])
	}
}

func TestGenerateRoundTripsToolHistory(t *testing.T) {
	var captured map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		err = json.Unmarshal(body, &captured)
		if err != nil {
			t.Fatal(err)
		}

		_, _ = w.Write([]byte(`{
			"model":"test",
			"message":{"role":"assistant","content":"The current UTC time is 2026-01-01T00:00:00Z."},
			"done":true,
			"done_reason":"stop"
		}`))
	}))
	defer srv.Close()

	model, err := ollamaadapter.New(
		ollamaadapter.WithModelName("test"),
		ollamaadapter.WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = model.Generate(context.Background(), intent.Request{
		Messages: []intent.Message{
			{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart("What time is it?")}},
			{
				Role: intent.RoleAI,
				ToolCalls: []intent.ToolCall{{
					ID:        "call_1",
					Type:      "function",
					Name:      "get_time",
					Arguments: `{}`,
				}},
			},
			{
				Role:       intent.RoleTool,
				Parts:      []intent.Part{intent.TextPart("2026-01-01T00:00:00Z")},
				ToolCallID: "call_1",
				ToolName:   "get_time",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	rawMessages, ok := captured["messages"].([]any)
	if !ok || len(rawMessages) != 3 {
		t.Fatalf("expected 3 messages, got %#v", captured["messages"])
	}

	toolMsg, ok := rawMessages[2].(map[string]any)
	if !ok {
		t.Fatalf("expected tool message map, got %#v", rawMessages[2])
	}

	if toolMsg["role"] != "tool" {
		t.Fatalf("expected tool role, got %#v", toolMsg["role"])
	}

	if toolMsg["tool_name"] != "get_time" {
		t.Fatalf("expected tool_name get_time, got %#v", toolMsg["tool_name"])
	}

	if !strings.Contains(toolMsg["content"].(string), "2026-01-01") {
		t.Fatalf("unexpected tool content: %#v", toolMsg["content"])
	}
}
