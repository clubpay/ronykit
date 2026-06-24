package intent

import (
	"context"
	"math/rand/v2"
	"sort"

	"github.com/clubpay/ronykit/intent/errs"
)

// SelectFirst returns the first model in the pool.
func SelectFirst(_ context.Context, models []LLM, _ Selection) (LLM, error) {
	if len(models) == 0 {
		return nil, errs.ErrEmptyPool
	}

	return models[0], nil
}

// SelectRandom returns a random model from the pool.
func SelectRandom(_ context.Context, models []LLM, _ Selection) (LLM, error) {
	if len(models) == 0 {
		return nil, errs.ErrEmptyPool
	}

	return models[rand.IntN(len(models))], nil
}

// SelectPriority returns the model with the highest Priority value.
func SelectPriority(_ context.Context, models []LLM, _ Selection) (LLM, error) {
	if len(models) == 0 {
		return nil, errs.ErrEmptyPool
	}

	best := models[0]

	bestPriority := best.Model().Priority
	for _, m := range models[1:] {
		priority := m.Model().Priority
		if priority > bestPriority {
			best = m
			bestPriority = priority
		}
	}

	return best, nil
}

// SelectEnforced returns the model matching sel.ModelID.
func SelectEnforced(_ context.Context, models []LLM, sel Selection) (LLM, error) {
	if sel.ModelID == "" {
		return nil, errs.Wrap(errs.ErrModelNotFound, "enforced selection requires model id")
	}

	for _, m := range models {
		model := m.Model()
		if model.ID == sel.ModelID || model.Name == sel.ModelID {
			return m, nil
		}
	}

	return nil, errs.ModelNotFound(sel.ModelID)
}

// Select dispatches to a built-in selector by strategy.
func Select(ctx context.Context, models []LLM, sel Selection) (LLM, error) {
	switch sel.Strategy {
	case StrategyRandom:
		return SelectRandom(ctx, models, sel)
	case StrategyPriority:
		return SelectPriority(ctx, models, sel)
	case StrategyEnforced:
		return SelectEnforced(ctx, models, sel)
	case StrategyFirst, "":
		return SelectFirst(ctx, models, sel)
	default:
		return nil, errs.Wrap(errs.ErrUnsupportedOperation, "unknown llm selection strategy")
	}
}

// ByPriority sorts models by descending priority.
func ByPriority(models []LLM) []LLM {
	out := append([]LLM(nil), models...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Model().Priority > out[j].Model().Priority
	})

	return out
}
