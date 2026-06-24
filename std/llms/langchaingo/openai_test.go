package langchaingo_test

import (
	"testing"

	"github.com/clubpay/ronykit/intent"
	lcadapter "github.com/clubpay/ronykit/std/llms/langchaingo"
)

func TestNewOpenAIDefaultInfo(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")

	model, err := lcadapter.NewOpenAI()
	if err != nil {
		t.Fatal(err)
	}

	info := model.Model()
	if info.ID != "openai" {
		t.Fatalf("id = %q", info.ID)
	}

	if info.Name != "gpt-4o-mini" {
		t.Fatalf("name = %q", info.Name)
	}
}

func TestNewOpenAICustomInfo(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")

	model, err := lcadapter.NewOpenAI(
		lcadapter.WithOpenAIModel("gpt-test"),
		lcadapter.WithOpenAIInfo(intent.Model{ID: "custom", Name: "Custom OpenAI"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	info := model.Model()
	if info.ID != "custom" || info.Name != "Custom OpenAI" {
		t.Fatalf("unexpected info: %#v", info)
	}
}

func TestMustNewOpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")

	model := lcadapter.MustNewOpenAI()
	if model.Model().ID != "openai" {
		t.Fatalf("id = %q", model.Model().ID)
	}
}
