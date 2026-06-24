package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultBaseURL   = "http://127.0.0.1:11434"
	toolTypeFunction = "function"
)

type client struct {
	baseURL    string
	httpClient *http.Client
}

func newClient(baseURL string, httpClient *http.Client) *client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

type chatRequest struct {
	Model    string         `json:"model"`
	Messages []chatMessage  `json:"messages"`
	Tools    []tool         `json:"tools,omitempty"`
	Stream   bool           `json:"stream"`
	Options  map[string]any `json:"options,omitempty"`
	Format   string         `json:"format,omitempty"`
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolName   string     `json:"tool_name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type tool struct {
	Type     string       `json:"type"`
	Function toolFunction `json:"function"`
}

type toolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"`
}

type toolCall struct {
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Index     int             `json:"index,omitempty"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type chatResponse struct {
	Model      string      `json:"model"`
	Message    chatMessage `json:"message"`
	Done       bool        `json:"done"`
	DoneReason string      `json:"done_reason,omitempty"`
}

func (c *client) chat(ctx context.Context, req chatRequest) (chatResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return chatResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return chatResponse{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return chatResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return chatResponse{}, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return chatResponse{}, fmt.Errorf("ollama chat failed: %s", strings.TrimSpace(string(respBody)))
	}

	var final chatResponse

	err = json.Unmarshal(respBody, &final)
	if err != nil {
		return chatResponse{}, fmt.Errorf("decode ollama chat response: %w", err)
	}

	if !final.Done {
		return chatResponse{}, fmt.Errorf("ollama chat returned incomplete response")
	}

	return final, nil
}
