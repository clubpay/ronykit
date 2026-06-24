package langchaingo

import (
	"fmt"
	"os"
	"strings"

	"github.com/clubpay/ronykit/intent"
	lcopenai "github.com/tmc/langchaingo/llms/openai"
)

const (
	EnvOpenAIModel   = "OPENAI_MODEL"
	EnvOpenAIAPIKey  = "OPENAI_API_KEY"
	EnvOpenAIBaseURL = "OPENAI_BASE_URL"

	DefaultOpenAIModel = "gpt-4o-mini"
)

type openaiConfig struct {
	modelName string
	apiKey    string
	baseURL   string
	info      intent.Model
}

// OpenAIOption configures an OpenAI-backed intent.LLM.
type OpenAIOption func(*openaiConfig)

// WithOpenAIModel sets the OpenAI model name.
func WithOpenAIModel(name string) OpenAIOption {
	return func(c *openaiConfig) {
		c.modelName = name
	}
}

// WithOpenAIAPIKey sets the OpenAI API key.
func WithOpenAIAPIKey(key string) OpenAIOption {
	return func(c *openaiConfig) {
		c.apiKey = key
	}
}

// WithOpenAIBaseURL sets a custom OpenAI-compatible base URL.
func WithOpenAIBaseURL(baseURL string) OpenAIOption {
	return func(c *openaiConfig) {
		c.baseURL = baseURL
	}
}

// WithOpenAIInfo sets model metadata exposed through intent.Model.
func WithOpenAIInfo(info intent.Model) OpenAIOption {
	return func(c *openaiConfig) {
		c.info = info
	}
}

func defaultOpenAIConfig() openaiConfig {
	return openaiConfig{
		modelName: envOr(EnvOpenAIModel, DefaultOpenAIModel),
		apiKey:    strings.TrimSpace(os.Getenv(EnvOpenAIAPIKey)),
		baseURL:   strings.TrimSpace(os.Getenv(EnvOpenAIBaseURL)),
		info:      intent.Model{ID: "openai", Priority: 3},
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}

	return fallback
}

// NewOpenAI returns an intent.LLM backed by OpenAI via langchaingo.
// Unset fields are populated from OPENAI_MODEL, OPENAI_API_KEY, and OPENAI_BASE_URL when present.
func NewOpenAI(opts ...OpenAIOption) (*Model, error) {
	cfg := defaultOpenAIConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	var lcOpts []lcopenai.Option

	if key := strings.TrimSpace(cfg.apiKey); key != "" {
		lcOpts = append(lcOpts, lcopenai.WithToken(key))
	}

	if baseURL := strings.TrimSpace(cfg.baseURL); baseURL != "" {
		lcOpts = append(lcOpts, lcopenai.WithBaseURL(baseURL))
	}

	modelName := strings.TrimSpace(cfg.modelName)
	if modelName != "" {
		lcOpts = append(lcOpts, lcopenai.WithModel(modelName))
	}

	raw, err := lcopenai.New(lcOpts...)
	if err != nil {
		return nil, fmt.Errorf("openai: %w", err)
	}

	info := cfg.info
	if info.ID == "" {
		info.ID = "openai"
	}

	if info.Name == "" {
		if modelName != "" {
			info.Name = modelName
		} else {
			info.Name = DefaultOpenAIModel
		}
	}

	return NewModel(raw, info), nil
}

// MustNewOpenAI is like NewOpenAI but panics on error.
func MustNewOpenAI(opts ...OpenAIOption) *Model {
	model, err := NewOpenAI(opts...)
	if err != nil {
		panic(err)
	}

	return model
}
