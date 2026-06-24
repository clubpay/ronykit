package intent

import "context"

// Strategy names built-in LLM selection behavior.
type Strategy string

const (
	StrategyRandom   Strategy = "random"
	StrategyFirst    Strategy = "first"
	StrategyPriority Strategy = "priority"
	StrategyEnforced Strategy = "enforced"
)

// Selection describes how to pick an LLM from a pool.
// ModelID is required when Strategy is StrategyEnforced.
type Selection struct {
	Strategy Strategy
	ModelID  string
}

// Selector picks one LLM from a pool.
type Selector interface {
	Select(ctx context.Context, models []LLM, sel Selection) (LLM, error)
}

// SelectorFunc adapts a function to Selector.
type SelectorFunc func(ctx context.Context, models []LLM, sel Selection) (LLM, error)

func (f SelectorFunc) Select(ctx context.Context, models []LLM, sel Selection) (LLM, error) {
	return f(ctx, models, sel)
}
