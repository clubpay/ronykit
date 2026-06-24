package ollama

import (
	"os"
	"strings"

	"github.com/clubpay/ronykit/intent"
)

type config struct {
	modelName string
	baseURL   string
	info      intent.Model
}

// Option configures an Ollama model adapter.
type Option func(*config)

// WithModelName sets the Ollama model name.
func WithModelName(name string) Option {
	return func(c *config) {
		c.modelName = name
	}
}

// WithBaseURL sets the Ollama server base URL.
func WithBaseURL(baseURL string) Option {
	return func(c *config) {
		c.baseURL = baseURL
	}
}

// WithInfo sets model metadata exposed through intent.Model.
func WithInfo(info intent.Model) Option {
	return func(c *config) {
		c.info = info
	}
}

func defaultConfig() config {
	return config{
		modelName: envOr(EnvModel, DefaultModel),
		baseURL:   envOr(EnvBaseURL, DefaultBaseURL),
		info:      intent.Model{ID: "ollama"},
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}

	return fallback
}
