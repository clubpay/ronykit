package decide

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/clubpay/ronykit/x/rkit"
)

type schemaConfig struct {
	resolver      Resolver
	model         string
	task          string
	lang          string
	minConfidence *float64
}

// SchemaOption configures a Schema[T].
type SchemaOption func(*schemaConfig)

// WithResolver sets the default Resolver used by Schema.Decide and Schema.DecideBatch.
func WithResolver(r Resolver) SchemaOption {
	return func(c *schemaConfig) {
		c.resolver = r
	}
}

// WithModel sets an optional model override sent in each Request.
func WithModel(model string) SchemaOption {
	return func(c *schemaConfig) {
		c.model = strings.TrimSpace(model)
	}
}

// WithTask sets an optional Laya routing task hint sent in each Request.
func WithTask(task string) SchemaOption {
	return func(c *schemaConfig) {
		c.task = strings.TrimSpace(task)
	}
}

// WithLang sets an optional Laya language hint sent in each Request.
func WithLang(lang string) SchemaOption {
	return func(c *schemaConfig) {
		c.lang = strings.TrimSpace(lang)
	}
}

// WithMinConfidence sets a schema-wide default minimum AnswerConfidence threshold.
// Individual fields can override this with their own .MinConfidence(...) call.
func WithMinConfidence(minConf float64) SchemaOption {
	return func(c *schemaConfig) {
		c.minConfidence = &minConf
	}
}

// DecideOption overrides settings for a single Decide or DecideBatch call.
type DecideOption func(*schemaConfig)

// WithCallResolver overrides the Schema's Resolver for a single Decide or DecideBatch call.
func WithCallResolver(r Resolver) DecideOption {
	return func(c *schemaConfig) {
		c.resolver = r
	}
}

// WithCallModel overrides the model for a single Decide or DecideBatch call.
func WithCallModel(model string) DecideOption {
	return func(c *schemaConfig) {
		c.model = strings.TrimSpace(model)
	}
}

// Schema is a compiled, reusable generic decision plan for struct type T.
// It converts T's bound fields into System 1 questions, executes them against a Resolver,
// and hydrates the returned answers into a strongly-typed Result[T].
type Schema[T any] struct {
	cfg       schemaConfig
	plans     []fieldPlan
	questions []Question
	byID      map[string]int
	byName    map[string]int
	byOffset  map[uintptr]int
}

// New creates and validates a compiled Schema[T] using build (and/or T's Describer[T] implementation).
func New[T any](build BuildFunc[T], opts ...SchemaOption) (*Schema[T], error) {
	return NewSchema[T](build, opts...)
}

// MustNew is like New but panics if the schema is invalid.
func MustNew[T any](build BuildFunc[T], opts ...SchemaOption) *Schema[T] {
	return rkit.Must(NewSchema[T](build, opts...))
}

// NewSchema creates and validates a compiled Schema[T].
// If T or *T implements Describer[T], its Describe method runs first; if build is non-nil,
// build runs next and may add or override field bindings.
func NewSchema[T any](build BuildFunc[T], opts ...SchemaOption) (*Schema[T], error) {
	var cfg schemaConfig
	for _, o := range opts {
		o(&cfg)
	}

	proto := new(T)

	b, err := newBuilder[T](proto)
	if err != nil {
		return nil, err
	}

	if d, ok := any(proto).(Describer[T]); ok {
		d.Describe(b)
	} else if d, ok := any(*proto).(Describer[T]); ok {
		d.Describe(b)
	}

	if build != nil {
		build(b, proto)
	}

	if len(b.errs) > 0 {
		return nil, b.errs[0]
	}

	if len(b.plans) == 0 {
		return nil, fmt.Errorf(
			"%w: schema for %v has no bound fields",
			ErrInvalidSchema,
			reflect.TypeFor[T](),
		)
	}

	if len(b.plans) > MaxQuestions {
		return nil, fmt.Errorf(
			"%w: schema has %d fields (max %d)",
			ErrInvalidSchema, len(b.plans), MaxQuestions,
		)
	}

	plans := make([]fieldPlan, len(b.plans))
	questions := make([]Question, len(b.plans))
	byID := make(map[string]int, len(b.plans))
	byName := make(map[string]int, len(b.plans)*2)
	byOffset := make(map[uintptr]int, len(b.plans))

	for i, p := range b.plans {
		if p.err != nil {
			return nil, p.err
		}

		if p.threshold <= 0 || p.threshold >= 1 {
			return nil, fmt.Errorf(
				"%w: field %s threshold %v must be in (0, 1)",
				ErrInvalidSchema, p.meta.namePath, p.threshold,
			)
		}

		if err := ValidateQuestion(p.question); err != nil {
			return nil, fmt.Errorf("field %s: %w", p.meta.namePath, err)
		}

		if _, dup := byID[p.question.ID]; dup {
			return nil, fmt.Errorf(
				"%w: duplicate question ID %q on field %s",
				ErrInvalidSchema, p.question.ID, p.meta.namePath,
			)
		}

		if p.minConfidence == nil && cfg.minConfidence != nil {
			mc := *cfg.minConfidence
			p.minConfidence = &mc
		}

		plans[i] = *p
		questions[i] = cloneQuestion(p.question)
		byID[p.question.ID] = i
		byName[p.meta.namePath] = i
		byName[strings.ToLower(p.meta.namePath)] = i
		byName[p.question.ID] = i
		byOffset[p.meta.offset] = i
	}

	return &Schema[T]{
		cfg:       cfg,
		plans:     plans,
		questions: questions,
		byID:      byID,
		byName:    byName,
		byOffset:  byOffset,
	}, nil
}

// MustSchema is like NewSchema but panics if the schema is invalid.
func MustSchema[T any](build BuildFunc[T], opts ...SchemaOption) *Schema[T] {
	return rkit.Must(NewSchema[T](build, opts...))
}

// NewSchemaFor creates a Schema[T] for a struct T (or *T) that implements Describer[T].
func NewSchemaFor[T any](opts ...SchemaOption) (*Schema[T], error) {
	return NewSchema[T](nil, opts...)
}

// MustSchemaFor is like NewSchemaFor but panics if the schema is invalid.
func MustSchemaFor[T any](opts ...SchemaOption) *Schema[T] {
	return rkit.Must(NewSchemaFor[T](opts...))
}

// Questions returns a copy of the compiled System 1 questions for this Schema.
func (s *Schema[T]) Questions() []Question {
	return rkit.Map(s.questions, cloneQuestion)
}

// Decide evaluates state against the Schema's questions using the configured Resolver
// and returns a populated Result[T].
func (s *Schema[T]) Decide(ctx context.Context, state any, opts ...DecideOption) (*Result[T], error) {
	cfg := s.cfg
	for _, o := range opts {
		o(&cfg)
	}

	if cfg.resolver == nil {
		return nil, ErrNoResolver
	}

	resolvedState, err := resolveStateInput(state)
	if err != nil {
		return nil, err
	}

	resp, err := cfg.resolver.Resolve(ctx, Request{
		State:         resolvedState,
		Questions:     s.Questions(),
		Model:         cfg.model,
		Task:          cfg.task,
		Lang:          cfg.lang,
		MinConfidence: cfg.minConfidence,
	})
	if err != nil {
		return nil, err
	}

	return s.hydrate(resp)
}

// DecideBatch evaluates multiple states against the Schema's questions.
// If the active Resolver implements BatchResolver (such as LayaResolver), all states
// are evaluated in a single batch call; otherwise Resolve is called sequentially per state.
func (s *Schema[T]) DecideBatch(
	ctx context.Context,
	states []any,
	opts ...DecideOption,
) ([]*Result[T], error) {
	cfg := s.cfg
	for _, o := range opts {
		o(&cfg)
	}

	if cfg.resolver == nil {
		return nil, ErrNoResolver
	}

	if len(states) == 0 {
		return []*Result[T]{}, nil
	}

	resolvedStates, err := resolveBatchStatesInput(states)
	if err != nil {
		return nil, err
	}

	if br, ok := cfg.resolver.(BatchResolver); ok {
		bResp, err := br.ResolveBatch(ctx, BatchRequest{
			States:        resolvedStates,
			Questions:     s.Questions(),
			Model:         cfg.model,
			Task:          cfg.task,
			Lang:          cfg.lang,
			MinConfidence: cfg.minConfidence,
		})
		if err != nil {
			return nil, err
		}

		if len(bResp.Results) != len(states) {
			return nil, fmt.Errorf(
				"%w: batch expected %d results, got %d",
				ErrInvalidResponse, len(states), len(bResp.Results),
			)
		}

		results := make([]*Result[T], len(states))

		for i, r := range bResp.Results {
			hydrated, err := s.hydrate(r)
			if err != nil {
				return nil, err
			}

			results[i] = hydrated
		}

		return results, nil
	}

	results := make([]*Result[T], len(resolvedStates))
	for i, st := range resolvedStates {
		res, err := s.Decide(ctx, st, opts...)
		if err != nil {
			return nil, err
		}

		results[i] = res
	}

	return results, nil
}

func (s *Schema[T]) hydrate(resp Response) (*Result[T], error) {
	var out T

	rv := reflect.ValueOf(&out).Elem()

	for i := range s.plans {
		p := &s.plans[i]

		ans, ok := resp.Answers[p.question.ID]
		if !ok {
			return nil, fmt.Errorf(
				"%w: missing answer for question %q (field %s)",
				ErrInvalidResponse, p.question.ID, p.meta.namePath,
			)
		}

		fv := fieldByIndexPath(rv, p.meta.indexPath)

		if p.minConfidence != nil && ans.AnswerConfidence() < *p.minConfidence {
			if p.defaultVal != nil {
				if err := assignConverted(fv, p.defaultVal); err != nil {
					return nil, fmt.Errorf("field %s default: %w", p.meta.namePath, err)
				}
			}

			continue
		}

		if err := hydrateField(fv, p, ans); err != nil {
			return nil, fmt.Errorf("field %s (%s): %w", p.meta.namePath, p.question.ID, err)
		}
	}

	return newResult(s, out, resp), nil
}

func fieldByIndexPath(rv reflect.Value, path []int) reflect.Value {
	curr := rv
	for _, idx := range path {
		if curr.Kind() == reflect.Pointer {
			if curr.IsNil() {
				curr.Set(reflect.New(curr.Type().Elem()))
			}

			curr = curr.Elem()
		}

		curr = curr.Field(idx)
	}

	return curr
}

func hydrateField(fv reflect.Value, p *fieldPlan, ans Answer) error {
	target := fv
	if target.Kind() == reflect.Pointer {
		alloc := reflect.New(derefType(target.Type()))
		if err := hydrateConcreteValue(alloc.Elem(), p, ans); err != nil {
			return err
		}

		// Wrap pointers if multiple levels exist (e.g. *string)
		ptrVal := alloc
		for ptrVal.Type() != target.Type() {
			nextPtr := reflect.New(ptrVal.Type())
			nextPtr.Elem().Set(ptrVal)
			ptrVal = nextPtr
		}

		target.Set(ptrVal)

		return nil
	}

	return hydrateConcreteValue(target, p, ans)
}

func hydrateConcreteValue(target reflect.Value, p *fieldPlan, ans Answer) error {
	switch p.question.Kind {
	case KindNoul:
		if ans.Noul == nil || math.IsNaN(*ans.Noul) {
			return fmt.Errorf("%w: noul value is missing", ErrInvalidResponse)
		}

		noul := *ans.Noul
		if p.rawProb {
			target.SetFloat(noul)

			return nil
		}

		target.SetBool(noul >= p.threshold)

		return nil

	case KindChoice:
		if ans.Choice == nil || *ans.Choice == "" {
			return fmt.Errorf("%w: choice value is missing", ErrInvalidResponse)
		}

		chosen := *ans.Choice
		if !hasOption(p.question.Options, chosen) {
			return fmt.Errorf("%w: choice %q is not in declared options", ErrInvalidResponse, chosen)
		}

		return assignStringOrInt(target, chosen)

	case KindScore:
		if ans.Score == nil && len(ans.Probabilities) == 0 {
			return fmt.Errorf("%w: score value is missing", ErrInvalidResponse)
		}

		if p.discreteScore || isIntKind(target.Kind()) {
			val := p.minLevel + ans.WinningLevelIndex(len(p.question.Levels))

			return assignInt(target, int64(val))
		}

		var rawScore float64
		if ans.Score != nil {
			rawScore = *ans.Score
		} else {
			rawScore = float64(ans.WinningLevelIndex(len(p.question.Levels)))
		}

		target.SetFloat(float64(p.minLevel) + rawScore)

		return nil

	default:
		return fmt.Errorf("%w: unsupported question kind %q", ErrInvalidQuestion, p.question.Kind)
	}
}

func hasOption(opts []ChoiceOption, label string) bool {
	for _, opt := range opts {
		if opt.Label == label {
			return true
		}
	}

	return false
}

func assignStringOrInt(target reflect.Value, label string) error {
	switch target.Kind() {
	case reflect.String:
		target.SetString(label)

		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(strings.TrimSpace(label), 10, 64)
		if err != nil {
			return fmt.Errorf("%w: cannot parse choice %q as int: %w", ErrInvalidResponse, label, err)
		}

		target.SetInt(n)

		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(strings.TrimSpace(label), 10, 64)
		if err != nil {
			return fmt.Errorf("%w: cannot parse choice %q as uint: %w", ErrInvalidResponse, label, err)
		}

		target.SetUint(n)

		return nil
	default:
		return fmt.Errorf("%w: unsupported choice target kind %v", ErrInvalidSchema, target.Kind())
	}
}

func assignInt(target reflect.Value, val int64) error {
	switch target.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		target.SetInt(val)

		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val < 0 {
			val = 0
		}

		target.SetUint(uint64(val))

		return nil
	case reflect.Float32, reflect.Float64:
		target.SetFloat(float64(val))

		return nil
	default:
		return fmt.Errorf("%w: unsupported score target kind %v", ErrInvalidSchema, target.Kind())
	}
}

func assignConverted(fv reflect.Value, val any) error {
	rv := reflect.ValueOf(val)
	if !rv.IsValid() {
		return nil
	}

	if fv.Kind() == reflect.Pointer && rv.Kind() != reflect.Pointer {
		ptr := reflect.New(fv.Type().Elem())
		if err := assignConverted(ptr.Elem(), val); err != nil {
			return err
		}

		fv.Set(ptr)

		return nil
	}

	if rv.Type().AssignableTo(fv.Type()) {
		fv.Set(rv)

		return nil
	}

	if rv.Type().ConvertibleTo(fv.Type()) {
		fv.Set(rv.Convert(fv.Type()))

		return nil
	}

	return fmt.Errorf("%w: cannot assign %T to %v", ErrInvalidSchema, val, fv.Type())
}

func cloneQuestion(q Question) Question {
	out := q
	if q.Options != nil {
		out.Options = slices.Clone(q.Options)
	}

	if q.Levels != nil {
		out.Levels = slices.Clone(q.Levels)
	}

	if q.NoulCriteria != nil {
		nc := *q.NoulCriteria
		out.NoulCriteria = &nc
	}

	return out
}
