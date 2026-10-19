package decide

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/clubpay/ronykit/x/rkit"
)

const (
	// DefaultLLMBaseURL is the default OpenAI-compatible API base URL.
	DefaultLLMBaseURL = "https://api.openai.com/v1"
	// DefaultLLMModel is the default chat completion model used by LLMResolver.
	DefaultLLMModel = "gpt-4o-mini"
)

const defaultLLMSystemPrompt = `You are a calibrated System 1 decision engine.
Evaluate the provided state against each question in the questions map and return a JSON object only.
You may return either:
1. {"answers": {"<id>": <answer>}} where each <answer> is:
   - for "noul": {"noul": <float in [0,1]>, "confidence": <float in [0,1]>} (or a boolean/number)
   - for "choice": {"choice": "<option_label>", "probabilities": {"<label>": <float>}, "confidence": <float>}
   - for "score": {"score": <number in [0, N-1]>, "probabilities": {"0": <float>}, "confidence": <float>}
2. Or a flat JSON object {"<id>": <value>} mapping each question ID directly to its
   boolean/probability (noul), option label (choice), or 0-based level number (score).`

type llmConfig struct {
	baseURL      string
	apiKey       string
	model        string
	systemPrompt string
	httpClient   *http.Client
}

// LLMOption configures an LLMResolver.
type LLMOption func(*llmConfig)

// WithLLMBaseURL sets the OpenAI-compatible base URL (e.g. "https://api.openai.com/v1").
func WithLLMBaseURL(baseURL string) LLMOption {
	return func(c *llmConfig) {
		c.baseURL = strings.TrimSpace(baseURL)
	}
}

// WithLLMAPIKey sets the Bearer API key for the LLM chat completions endpoint.
func WithLLMAPIKey(apiKey string) LLMOption {
	return func(c *llmConfig) {
		c.apiKey = strings.TrimSpace(apiKey)
	}
}

// WithLLMModel sets the default LLM model name.
func WithLLMModel(model string) LLMOption {
	return func(c *llmConfig) {
		c.model = strings.TrimSpace(model)
	}
}

// WithLLMSystemPrompt overrides the default System 1 instructions prompt.
func WithLLMSystemPrompt(prompt string) LLMOption {
	return func(c *llmConfig) {
		c.systemPrompt = strings.TrimSpace(prompt)
	}
}

// WithLLMHTTPClient sets a custom *http.Client for LLM requests.
func WithLLMHTTPClient(client *http.Client) LLMOption {
	return func(c *llmConfig) {
		c.httpClient = client
	}
}

// LLMResolver evaluates System 1 questions using an OpenAI-compatible `/chat/completions` endpoint.
type LLMResolver struct {
	cfg llmConfig
}

// NewLLMResolver creates a new LLMResolver.
// Unset configuration fields fall back to environment variables:
//   - DECIDE_LLM_BASE_URL (defaults to DefaultLLMBaseURL)
//   - DECIDE_LLM_API_KEY (or OPENAI_API_KEY)
//   - DECIDE_LLM_MODEL (defaults to DefaultLLMModel)
func NewLLMResolver(opts ...LLMOption) (*LLMResolver, error) {
	cfg := llmConfig{
		baseURL: strings.TrimSpace(os.Getenv("DECIDE_LLM_BASE_URL")),
		apiKey: rkit.Coalesce(
			strings.TrimSpace(os.Getenv("DECIDE_LLM_API_KEY")),
			strings.TrimSpace(os.Getenv("OPENAI_API_KEY")),
		),
		model:        strings.TrimSpace(os.Getenv("DECIDE_LLM_MODEL")),
		systemPrompt: defaultLLMSystemPrompt,
		httpClient:   defaultHTTPClient,
	}

	for _, o := range opts {
		o(&cfg)
	}

	cfg.baseURL = rkit.Coalesce(cfg.baseURL, DefaultLLMBaseURL)
	cfg.model = rkit.Coalesce(cfg.model, DefaultLLMModel)
	cfg.systemPrompt = rkit.Coalesce(cfg.systemPrompt, defaultLLMSystemPrompt)

	cfg.httpClient = rkit.Coalesce(cfg.httpClient, defaultHTTPClient)

	return &LLMResolver{cfg: cfg}, nil
}

// MustNewLLMResolver is like NewLLMResolver but panics on error.
func MustNewLLMResolver(opts ...LLMOption) *LLMResolver {
	return rkit.Must(NewLLMResolver(opts...))
}

// Resolve implements Resolver.
func (r *LLMResolver) Resolve(ctx context.Context, req Request) (Response, error) {
	questionsWire, err := buildSystemOneQuestions(req.Questions)
	if err != nil {
		return Response{}, err
	}

	resolvedState, err := resolveStateInput(req.State)
	if err != nil {
		return Response{}, err
	}

	model := rkit.Coalesce(strings.TrimSpace(req.Model), r.cfg.model, DefaultLLMModel)

	userPayload := map[string]any{
		"state":     resolvedState,
		"questions": questionsWire,
	}

	chatReq := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": r.cfg.systemPrompt},
			{"role": "user", "content": rkit.ToJSONStr(userPayload)},
		},
		"temperature":     0,
		"response_format": map[string]string{"type": "json_object"},
	}

	rawBody, err := json.Marshal(chatReq)
	if err != nil {
		return Response{}, err
	}

	url := strings.TrimRight(r.cfg.baseURL, "/")
	if !strings.HasSuffix(url, "/chat/completions") {
		url += "/chat/completions"
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(rawBody))
	if err != nil {
		return Response{}, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	if r.cfg.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+r.cfg.apiKey)
	}

	resp, err := r.cfg.httpClient.Do(httpReq)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("%w: read body: %w", ErrUnavailable, err)
	}

	if err := classifyHTTPStatus(resp.StatusCode, payload); err != nil {
		return Response{}, err
	}

	return parseLLMChatCompletion(req.Questions, model, payload)
}

type chatCompletionEnvelope struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func parseLLMChatCompletion(questions []Question, fallbackModel string, payload []byte) (Response, error) {
	var env chatCompletionEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return Response{}, fmt.Errorf("%w: decode chat completion: %w", ErrInvalidResponse, err)
	}

	if len(env.Choices) == 0 {
		return Response{}, fmt.Errorf("%w: empty chat completion choices", ErrInvalidResponse)
	}

	content := stripMarkdownCodeFence(strings.TrimSpace(env.Choices[0].Message.Content))

	var rawMap map[string]any
	if err := json.Unmarshal(rkit.S2B(content), &rawMap); err != nil {
		return Response{}, fmt.Errorf("%w: decode llm json content: %w", ErrInvalidResponse, err)
	}

	// Support both {"answers": {...}} and flat {"<id>": ...}
	answersMap := rawMap
	if nested, ok := rawMap["answers"].(map[string]any); ok {
		answersMap = nested
	}

	answers := make(map[string]Answer, len(questions))
	for _, q := range questions {
		entry, ok := answersMap[q.ID]
		if !ok {
			return Response{}, fmt.Errorf("%w: llm answer missing for question %q", ErrInvalidResponse, q.ID)
		}

		ans, err := coerceLLMAnswer(q, entry)
		if err != nil {
			return Response{}, fmt.Errorf("question %q: %w", q.ID, err)
		}

		answers[q.ID] = ans
	}

	return Response{
		Model:   rkit.Coalesce(env.Model, fallbackModel),
		Answers: answers,
		Usage: Usage{
			InputTokens:  env.Usage.PromptTokens,
			OutputTokens: env.Usage.CompletionTokens,
		},
	}, nil
}

func coerceLLMAnswer(q Question, entry any) (Answer, error) {
	if m, ok := entry.(map[string]any); ok {
		return coerceLLMObjectAnswer(q, m)
	}

	conf := 1.0

	switch q.Kind {
	case KindNoul:
		var noul float64

		switch v := entry.(type) {
		case bool:
			if v {
				noul = 1.0
			} else {
				noul = 0.0
			}
		default:
			f, ok := parseFloatAny(v)
			if !ok {
				return Answer{}, fmt.Errorf(
					"%w: expected bool or float for noul, got %T",
					ErrInvalidResponse,
					entry,
				)
			}

			noul = f
		}

		return Answer{
			Kind:       KindNoul,
			Noul:       &noul,
			Confidence: &conf,
			Probabilities: map[string]float64{
				LabelTrue:  noul,
				LabelFalse: 1 - noul,
			},
		}, nil

	case KindChoice:
		choice, ok := parseChoiceAny(entry)
		if !ok || choice == "" {
			return Answer{}, fmt.Errorf("%w: expected string/int choice, got %T", ErrInvalidResponse, entry)
		}

		probs := make(map[string]float64, len(q.Options))
		for _, opt := range q.Options {
			if opt.Label == choice {
				probs[opt.Label] = 1.0
			} else {
				probs[opt.Label] = 0.0
			}
		}

		return Answer{
			Kind:          KindChoice,
			Choice:        &choice,
			Probabilities: probs,
			Confidence:    &conf,
		}, nil

	case KindScore:
		score, ok := parseFloatAny(entry)
		if !ok {
			return Answer{}, fmt.Errorf("%w: expected numeric score, got %T", ErrInvalidResponse, entry)
		}

		return Answer{
			Kind:       KindScore,
			Score:      &score,
			Confidence: &conf,
		}, nil

	default:
		return Answer{}, fmt.Errorf("%w: unsupported question kind %q", ErrInvalidQuestion, q.Kind)
	}
}

func coerceLLMObjectAnswer(q Question, m map[string]any) (Answer, error) {
	ans := Answer{
		Kind: q.Kind,
	}

	if c, ok := parseFloatAny(m["confidence"]); ok {
		ans.Confidence = &c
	}

	if rawProbs, ok := m["probabilities"].(map[string]any); ok {
		probs := make(map[string]float64, len(rawProbs))
		for k, v := range rawProbs {
			if f, ok := parseFloatAny(v); ok {
				probs[k] = f
			}
		}

		ans.Probabilities = probs
	}

	switch q.Kind {
	case KindNoul:
		val := m["noul"]
		if b, ok := val.(bool); ok {
			f := 0.0
			if b {
				f = 1.0
			}

			ans.Noul = &f

			return ans, nil
		}

		f, ok := parseFloatAny(val)
		if !ok {
			return Answer{}, fmt.Errorf("%w: llm noul field is not a number or bool", ErrInvalidResponse)
		}

		ans.Noul = &f

		return ans, nil

	case KindChoice:
		choice, ok := parseChoiceAny(m["choice"])
		if !ok || choice == "" {
			return Answer{}, fmt.Errorf("%w: llm choice field is missing", ErrInvalidResponse)
		}

		ans.Choice = &choice

		return ans, nil

	case KindScore:
		score, ok := parseFloatAny(m["score"])
		if !ok {
			return Answer{}, fmt.Errorf("%w: llm score field is not a number", ErrInvalidResponse)
		}

		ans.Score = &score

		return ans, nil

	default:
		return Answer{}, fmt.Errorf("%w: unsupported question kind %q", ErrInvalidQuestion, q.Kind)
	}
}

func stripMarkdownCodeFence(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}

	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[idx+1:]
	}

	s = strings.TrimSuffix(strings.TrimSpace(s), "```")

	return strings.TrimSpace(s)
}
