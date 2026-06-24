package intent

import (
	"context"

	"github.com/clubpay/ronykit/intent/errs"
)

// DefaultLLMPool is a Pool backed by a Selector.
type DefaultLLMPool struct {
	models   []LLM
	selector Selector
}

// NewLLMPool returns a pool with the given models and selector.
// When the selector is nil, StrategyFirst is used.
func NewLLMPool(models []LLM, selector Selector) (*DefaultLLMPool, error) {
	if len(models) == 0 {
		return nil, errs.ErrEmptyPool
	}

	sel := selector
	if sel == nil {
		sel = SelectorFunc(SelectFirst)
	}

	return &DefaultLLMPool{
		models:   append([]LLM(nil), models...),
		selector: sel,
	}, nil
}

func (p *DefaultLLMPool) Models() []Model {
	out := make([]Model, 0, len(p.models))
	for _, m := range p.models {
		out = append(out, m.Model())
	}

	return out
}

func (p *DefaultLLMPool) Select(ctx context.Context, sel Selection) (LLM, error) {
	if p == nil || len(p.models) == 0 {
		return nil, errs.ErrEmptyPool
	}

	if sel.Strategy == "" {
		sel.Strategy = StrategyFirst
	}

	return p.selector.Select(ctx, p.models, sel)
}
