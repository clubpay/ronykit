package decide

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/clubpay/ronykit/x/rkit"
)

var (
	// ErrUnavailable indicates a transient transport, rate-limit (429), or 5xx error from a resolver.
	ErrUnavailable = errors.New("decide: resolver unavailable")
	// ErrInvalidSchema indicates a schema definition or field binding error.
	ErrInvalidSchema = errors.New("decide: invalid schema")
	// ErrInvalidQuestion indicates a malformed question definition.
	ErrInvalidQuestion = errors.New("decide: invalid question")
	// ErrInvalidResponse indicates the resolver returned a malformed or incomplete answer payload.
	ErrInvalidResponse = errors.New("decide: invalid response")
	// ErrNoResolver indicates Decide was called without a configured Resolver.
	ErrNoResolver = errors.New("decide: resolver is not configured")
)

// Resolver evaluates a Question set against a State.
type Resolver interface {
	Resolve(ctx context.Context, req Request) (Response, error)
}

// BatchResolver is an optional extension implemented by resolvers that support
// evaluating multiple states in a single batch call (such as Laya).
// When a Resolver does not implement BatchResolver, Schema.DecideBatch falls back
// to calling Resolve per state.
type BatchResolver interface {
	Resolver
	ResolveBatch(ctx context.Context, req BatchRequest) (BatchResponse, error)
}

// ResolverFunc adapts a function to the Resolver interface.
type ResolverFunc func(ctx context.Context, req Request) (Response, error)

// Resolve implements Resolver.
func (f ResolverFunc) Resolve(ctx context.Context, req Request) (Response, error) {
	return f(ctx, req)
}

// ValidateQuestion checks that a Question satisfies System 1 constraints.
func ValidateQuestion(q Question) error {
	if q.ID == "" {
		return fmt.Errorf("%w: question ID must not be empty", ErrInvalidQuestion)
	}

	switch q.Kind {
	case KindNoul:
		return nil
	case KindChoice:
		if len(q.Options) == 0 {
			return fmt.Errorf(
				"%w: choice question %q must have at least one option",
				ErrInvalidQuestion,
				q.ID,
			)
		}

		if len(q.Options) > MaxChoiceOptions {
			return fmt.Errorf(
				"%w: choice question %q has %d options (max %d)",
				ErrInvalidQuestion, q.ID, len(q.Options), MaxChoiceOptions,
			)
		}

		seen := make(map[string]struct{}, len(q.Options))
		for _, opt := range q.Options {
			if opt.Label == "" {
				return fmt.Errorf(
					"%w: choice question %q has an empty option label",
					ErrInvalidQuestion,
					q.ID,
				)
			}

			if _, dup := seen[opt.Label]; dup {
				return fmt.Errorf(
					"%w: choice question %q has duplicate option label %q",
					ErrInvalidQuestion, q.ID, opt.Label,
				)
			}

			seen[opt.Label] = struct{}{}
		}

		return nil
	case KindScore:
		if len(q.Levels) < MinScoreLevels || len(q.Levels) > MaxScoreLevels {
			return fmt.Errorf(
				"%w: score question %q must have between %d and %d levels, got %d",
				ErrInvalidQuestion, q.ID, MinScoreLevels, MaxScoreLevels, len(q.Levels),
			)
		}

		return nil
	default:
		return fmt.Errorf("%w: question %q has unknown kind %q", ErrInvalidQuestion, q.ID, q.Kind)
	}
}

type passResolver struct{}

// NewPassResolver returns a deterministic Resolver for local development and tests.
// It answers KindNoul with 1.0 (true), KindChoice with the first declared option,
// and KindScore with 0.0 (first level).
func NewPassResolver() Resolver {
	return passResolver{}
}

func (passResolver) Resolve(_ context.Context, req Request) (Response, error) {
	answers := make(map[string]Answer, len(req.Questions))

	for _, q := range req.Questions {
		if err := ValidateQuestion(q); err != nil {
			return Response{}, err
		}

		conf := 1.0

		switch q.Kind {
		case KindNoul:
			noul := 1.0
			answers[q.ID] = Answer{
				Kind:       KindNoul,
				Noul:       &noul,
				Confidence: &conf,
				Probabilities: map[string]float64{
					LabelTrue:  1.0,
					LabelFalse: 0.0,
				},
			}
		case KindChoice:
			chosen := q.Options[0].Label

			probs := make(map[string]float64, len(q.Options))
			for i, opt := range q.Options {
				if i == 0 {
					probs[opt.Label] = 1.0
				} else {
					probs[opt.Label] = 0.0
				}
			}

			answers[q.ID] = Answer{
				Kind:          KindChoice,
				Choice:        &chosen,
				Probabilities: probs,
				Confidence:    &conf,
			}
		case KindScore:
			score := 0.0
			probs := make(map[string]float64, len(q.Levels))
			legend := make(map[string]string, len(q.Levels))

			for i, lvl := range q.Levels {
				k := rkit.IntToStr(i)
				legend[k] = lvl

				if i == 0 {
					probs[k] = 1.0
				} else {
					probs[k] = 0.0
				}
			}

			answers[q.ID] = Answer{
				Kind:          KindScore,
				Score:         &score,
				Probabilities: probs,
				Legend:        legend,
				Confidence:    &conf,
			}
		}
	}

	return Response{
		Model:   rkit.Coalesce(req.Model, "pass"),
		Answers: answers,
	}, nil
}

type staticResolver struct {
	model   string
	answers map[string]Answer
}

// NewStaticResolver returns a Resolver that replies with a fixed map of answers keyed by question ID.
// Questions not present in the static map fall back to NewPassResolver behavior.
func NewStaticResolver(answers map[string]Answer) Resolver {
	return staticResolver{
		model:   "static",
		answers: maps.Clone(answers),
	}
}

func (s staticResolver) Resolve(ctx context.Context, req Request) (Response, error) {
	base, err := NewPassResolver().Resolve(ctx, req)
	if err != nil {
		return Response{}, err
	}

	base.Model = rkit.Coalesce(req.Model, s.model)

	for _, q := range req.Questions {
		if ans, ok := s.answers[q.ID]; ok {
			base.Answers[q.ID] = cloneAnswer(ans)
		}
	}

	return base, nil
}

func cloneAnswer(a Answer) Answer {
	out := a
	if a.Probabilities != nil {
		out.Probabilities = maps.Clone(a.Probabilities)
	}

	if a.Legend != nil {
		out.Legend = maps.Clone(a.Legend)
	}

	return out
}

func intToStr(v int) string {
	return rkit.IntToStr(v)
}
