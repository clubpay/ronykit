package intent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/intent/errs"
)

type stubLLM struct {
	info intent.Model
}

func (s stubLLM) Model() intent.Model { return s.info }

func (s stubLLM) Generate(context.Context, intent.Request) (intent.Response, error) {
	return intent.Response{}, nil
}

func (s stubLLM) Stream(context.Context, intent.Request) (intent.Stream, error) {
	return nil, nil
}

func TestNewPool_Empty(t *testing.T) {
	_, err := intent.NewLLMPool(nil, nil)
	if !errors.Is(err, errs.ErrEmptyPool) {
		t.Fatalf("expected ErrEmptyPool, got %v", err)
	}
}

func TestSelectFirst(t *testing.T) {
	models := []intent.LLM{
		stubLLM{info: intent.Model{ID: "a"}},
		stubLLM{info: intent.Model{ID: "b"}},
	}

	got, err := intent.SelectFirst(context.Background(), models, intent.Selection{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Model().ID != "a" {
		t.Fatalf("got %q, want a", got.Model().ID)
	}
}

func TestSelectEnforcedMissingID(t *testing.T) {
	_, err := intent.SelectEnforced(context.Background(), []intent.LLM{stubLLM{info: intent.Model{ID: "a"}}}, intent.Selection{
		Strategy: intent.StrategyEnforced,
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSelectEnforcedFound(t *testing.T) {
	models := []intent.LLM{
		stubLLM{info: intent.Model{ID: "a"}},
		stubLLM{info: intent.Model{ID: "b"}},
	}

	got, err := intent.SelectEnforced(context.Background(), models, intent.Selection{
		Strategy: intent.StrategyEnforced,
		ModelID:  "b",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Model().ID != "b" {
		t.Fatalf("got %q, want b", got.Model().ID)
	}
}

func TestSelectPriority(t *testing.T) {
	models := []intent.LLM{
		stubLLM{info: intent.Model{ID: "low", Priority: 1}},
		stubLLM{info: intent.Model{ID: "high", Priority: 9}},
	}

	got, err := intent.SelectPriority(context.Background(), models, intent.Selection{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Model().ID != "high" {
		t.Fatalf("got %q, want high", got.Model().ID)
	}
}

func TestDefaultPoolSelect(t *testing.T) {
	pool, err := intent.NewLLMPool([]intent.LLM{
		stubLLM{info: intent.Model{ID: "only"}},
	}, intent.SelectorFunc(intent.SelectFirst))
	if err != nil {
		t.Fatal(err)
	}

	got, err := pool.Select(context.Background(), intent.Selection{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Model().ID != "only" {
		t.Fatalf("got %q, want only", got.Model().ID)
	}
}
