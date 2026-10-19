package decide

import (
	"bytes"
	"fmt"
	"maps"
	"strings"
	"text/template"

	"github.com/clubpay/ronykit/x/rkit"
)

// StateEvaluator is implemented by lazy or template-backed state values that render
// or compute their underlying state payload immediately before a Resolver is invoked.
type StateEvaluator interface {
	EvalState() (any, error)
}

var defaultTemplateFuncs = template.FuncMap{
	"json":    rkit.ToJSONStr,
	"trim":    strings.TrimSpace,
	"upper":   strings.ToUpper,
	"lower":   strings.ToLower,
	"join":    strings.Join,
	"snake":   rkit.ToSnake,
	"camel":   rkit.ToCamel,
	"kebab":   rkit.ToKebab,
	"default": func(def, val string) string { return rkit.Coalesce(val, def) },
}

type templateConfig struct {
	name       string
	missingKey string
	funcs      template.FuncMap
}

// TemplateOption configures a StateTemplate.
type TemplateOption func(*templateConfig)

// WithTemplateName sets the template name used in parse/execution diagnostics.
func WithTemplateName(name string) TemplateOption {
	return func(c *templateConfig) {
		c.name = strings.TrimSpace(name)
	}
}

// WithTemplateFuncs registers custom template functions (merged on top of the built-in
// `json`, `trim`, `upper`, `lower`, `join`, `snake`, `camel`, `kebab`, and `default` helpers).
func WithTemplateFuncs(funcs template.FuncMap) TemplateOption {
	return func(c *templateConfig) {
		if c.funcs == nil {
			c.funcs = make(template.FuncMap, len(funcs))
		}

		maps.Copy(c.funcs, funcs)
	}
}

// WithMissingKey sets the text/template missingkey option (e.g. "error", "zero", "default").
func WithMissingKey(mode string) TemplateOption {
	return func(c *templateConfig) {
		c.missingKey = strings.TrimSpace(mode)
	}
}

// StateTemplate is a compiled Go text/template that renders typed data into state text.
// It is safe for concurrent use across goroutines.
type StateTemplate[IN any] struct {
	tmpl *template.Template
}

// NewTemplate compiles a Go text/template string for input type IN.
// Built-in template functions available in tmplText:
//   - `json`: marshals any value as JSON
//   - `trim`, `upper`, `lower`, `join`
//   - `snake`, `camel`, `kebab`
//   - `default`: `{{ default "fallback" .Val }}`
func NewTemplate[IN any](tmplText string, opts ...TemplateOption) (*StateTemplate[IN], error) {
	cfg := templateConfig{
		name: "decide_state",
	}

	for _, o := range opts {
		o(&cfg)
	}

	funcs := make(template.FuncMap, len(defaultTemplateFuncs)+len(cfg.funcs))
	maps.Copy(funcs, defaultTemplateFuncs)

	maps.Copy(funcs, cfg.funcs)

	t := template.New(rkit.Coalesce(cfg.name, "decide_state")).Funcs(funcs)
	if cfg.missingKey != "" {
		t = t.Option("missingkey=" + cfg.missingKey)
	}

	parsed, err := t.Parse(tmplText)
	if err != nil {
		return nil, fmt.Errorf("%w: parse state template: %w", ErrInvalidSchema, err)
	}

	return &StateTemplate[IN]{tmpl: parsed}, nil
}

// MustTemplate is like NewTemplate but panics if the template fails to parse.
func MustTemplate[IN any](tmplText string, opts ...TemplateOption) *StateTemplate[IN] {
	return rkit.Must(NewTemplate[IN](tmplText, opts...))
}

// Render executes the compiled template with data and returns the rendered state string.
func (st *StateTemplate[IN]) Render(data IN) (string, error) {
	var buf bytes.Buffer
	if err := st.tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("%w: render state template: %w", ErrInvalidSchema, err)
	}

	return strings.TrimSpace(buf.String()), nil
}

// MustRender executes the compiled template with data and panics on error.
func (st *StateTemplate[IN]) MustRender(data IN) string {
	return rkit.Must(st.Render(data))
}

// State returns a lazy StateEvaluator that renders data through st when passed to Schema.Decide
// or Schema.DecideBatch.
func (st *StateTemplate[IN]) State(data IN) StateEvaluator {
	return templateStateValue[IN]{
		tmpl: st,
		data: data,
	}
}

// States converts a slice of typed inputs into a []any slice of lazy StateEvaluators
// suitable for passing directly to Schema.DecideBatch.
func (st *StateTemplate[IN]) States(items []IN) []any {
	return rkit.Map(items, func(item IN) any {
		return st.State(item)
	})
}

type templateStateValue[IN any] struct {
	tmpl *StateTemplate[IN]
	data IN
}

func (v templateStateValue[IN]) EvalState() (any, error) {
	return v.tmpl.Render(v.data)
}

type rawTemplateStateValue[IN any] struct {
	tmplText string
	data     IN
	opts     []TemplateOption
}

func (v rawTemplateStateValue[IN]) EvalState() (any, error) {
	st, err := NewTemplate[IN](v.tmplText, v.opts...)
	if err != nil {
		return nil, err
	}

	return st.Render(v.data)
}

// TemplateState creates a StateEvaluator that renders tmplText with data (a struct, map, or primitive)
// when passed to Schema.Decide or Schema.DecideBatch.
// For templates executed repeatedly, prefer compiling once with NewTemplate / MustTemplate.
func TemplateState[IN any](tmplText string, data IN, opts ...TemplateOption) StateEvaluator {
	return rawTemplateStateValue[IN]{
		tmplText: tmplText,
		data:     data,
		opts:     opts,
	}
}

// RenderState parses tmplText and immediately renders it with data.
func RenderState[IN any](tmplText string, data IN, opts ...TemplateOption) (string, error) {
	st, err := NewTemplate[IN](tmplText, opts...)
	if err != nil {
		return "", err
	}

	return st.Render(data)
}

// MustRenderState is like RenderState but panics on error.
func MustRenderState[IN any](tmplText string, data IN, opts ...TemplateOption) string {
	return rkit.Must(RenderState(tmplText, data, opts...))
}

func resolveStateInput(state any) (any, error) {
	if eval, ok := state.(StateEvaluator); ok {
		return eval.EvalState()
	}

	return state, nil
}

func resolveBatchStatesInput(states []any) ([]any, error) {
	out := make([]any, len(states))
	for i, s := range states {
		resolved, err := resolveStateInput(s)
		if err != nil {
			return nil, fmt.Errorf("batch state %d: %w", i, err)
		}

		out[i] = resolved
	}

	return out, nil
}
