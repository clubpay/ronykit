package decide

import (
	"maps"
	"reflect"
	"strings"
	"unsafe"
)

// Result holds the hydrated struct Value of type T along with per-field probabilities,
// confidence scores, model identifier, and token usage returned by the Resolver.
type Result[T any] struct {
	schema *Schema[T]
	value  T
	resp   Response
}

func newResult[T any](schema *Schema[T], value T, resp Response) *Result[T] {
	return &Result[T]{
		schema: schema,
		value:  value,
		resp:   resp,
	}
}

// Value returns the populated struct T.
func (r *Result[T]) Value() T {
	return r.value
}

// Ptr returns a pointer to the internal populated struct T.
// Field pointers taken from Ptr() (e.g. &res.Ptr().Department) can be passed directly
// to ProbabilitiesOf, ConfidenceOf, AnswerConfidenceOf, NoulOf, ScoreOf, and AnswerOf.
func (r *Result[T]) Ptr() *T {
	return &r.value
}

// Model returns the model or checkpoint identifier that produced the answers.
func (r *Result[T]) Model() string {
	return r.resp.Model
}

// Usage returns the token usage and truncation report for the decision call.
func (r *Result[T]) Usage() Usage {
	return r.resp.Usage
}

// Routing returns Laya's router metadata when present, or nil.
func (r *Result[T]) Routing() *RoutingInfo {
	return r.resp.Routing
}

// RawResponse returns the underlying Resolver Response.
func (r *Result[T]) RawResponse() Response {
	return r.resp
}

// Answers returns all raw Answers keyed by question ID.
func (r *Result[T]) Answers() map[string]Answer {
	out := make(map[string]Answer, len(r.resp.Answers))
	for k, v := range r.resp.Answers {
		out[k] = cloneAnswer(v)
	}

	return out
}

// Probabilities returns the option/level probability distributions for all questions,
// keyed by question ID.
// For KindScore questions with defined Levels, keys are the level indices ("0", "1", ...);
// for KindChoice questions, keys are the option labels; for KindNoul questions without
// explicit probabilities, {"true": noul, "false": 1 - noul} is synthesized.
func (r *Result[T]) Probabilities() map[string]map[string]float64 {
	out := make(map[string]map[string]float64, len(r.schema.plans))
	for i := range r.schema.plans {
		p := &r.schema.plans[i]
		if probs := r.probabilitiesForPlan(p); len(probs) > 0 {
			out[p.question.ID] = probs
		}
	}

	return out
}

// Confidence returns the reported confidence scores for all questions keyed by question ID.
// When a backend omits `confidence` on a KindNoul answer (such as Jev), AnswerConfidence()
// (`max(noul, 1-noul)`) is returned so callers always receive a usable value.
func (r *Result[T]) Confidence() map[string]float64 {
	out := make(map[string]float64, len(r.schema.plans))
	for i := range r.schema.plans {
		p := &r.schema.plans[i]
		if ans, ok := r.resp.Answers[p.question.ID]; ok {
			if ans.Confidence != nil {
				out[p.question.ID] = *ans.Confidence
			} else {
				out[p.question.ID] = ans.AnswerConfidence()
			}
		}
	}

	return out
}

// ProbabilitiesOf returns the probability map for a specific field.
// The field argument may be:
//   - a field pointer into res.Ptr(), e.g. `&res.Ptr().Department`
//   - a selector function, e.g. `func(t *Ticket) any { return &t.Department }`
//   - a field name ("Department") or question ID ("department")
func (r *Result[T]) ProbabilitiesOf(field any) map[string]float64 {
	p, ok := r.resolvePlan(field)
	if !ok {
		return nil
	}

	return r.probabilitiesForPlan(p)
}

// ConfidenceOf returns the confidence score for a specific field (or 0 if not found).
// Accepts a field pointer into res.Ptr(), a selector `func(*T) any`, or a field/question name.
func (r *Result[T]) ConfidenceOf(field any) float64 {
	ans, ok := r.AnswerOf(field)
	if !ok {
		return 0
	}

	if ans.Confidence != nil {
		return *ans.Confidence
	}

	return ans.AnswerConfidence()
}

// AnswerConfidenceOf returns max(p) across options/sides for a specific field.
// Accepts a field pointer into res.Ptr(), a selector `func(*T) any`, or a field/question name.
func (r *Result[T]) AnswerConfidenceOf(field any) float64 {
	ans, ok := r.AnswerOf(field)
	if !ok {
		return 0
	}

	return ans.AnswerConfidence()
}

// NoulOf returns the raw noul probability in [0, 1] for a Bool or Prob field (or 0 if not found).
// Accepts a field pointer into res.Ptr(), a selector `func(*T) any`, or a field/question name.
func (r *Result[T]) NoulOf(field any) float64 {
	ans, ok := r.AnswerOf(field)
	if !ok || ans.Noul == nil {
		return 0
	}

	return *ans.Noul
}

// ScoreOf returns the raw probability-weighted score for a Score field (or 0 if not found).
// Accepts a field pointer into res.Ptr(), a selector `func(*T) any`, or a field/question name.
func (r *Result[T]) ScoreOf(field any) float64 {
	ans, ok := r.AnswerOf(field)
	if !ok || ans.Score == nil {
		return 0
	}

	return *ans.Score
}

// AnswerOf returns the raw Answer and true for a specific field, or (Answer{}, false) if unknown.
// Accepts a field pointer into res.Ptr(), a selector `func(*T) any`, or a field/question name.
func (r *Result[T]) AnswerOf(field any) (Answer, bool) {
	p, ok := r.resolvePlan(field)
	if !ok {
		return Answer{}, false
	}

	ans, ok := r.resp.Answers[p.question.ID]
	if !ok {
		return Answer{}, false
	}

	return cloneAnswer(ans), true
}

func (r *Result[T]) probabilitiesForPlan(p *fieldPlan) map[string]float64 {
	ans, ok := r.resp.Answers[p.question.ID]
	if !ok {
		return nil
	}

	if len(ans.Probabilities) > 0 {
		return maps.Clone(ans.Probabilities)
	}

	if ans.Noul != nil {
		noul := *ans.Noul

		return map[string]float64{
			LabelTrue:  noul,
			LabelFalse: 1 - noul,
		}
	}

	return nil
}

func (r *Result[T]) resolvePlan(field any) (*fieldPlan, bool) {
	if field == nil || r.schema == nil {
		return nil, false
	}

	switch f := field.(type) {
	case string:
		key := strings.TrimSpace(f)
		if idx, ok := r.schema.byName[key]; ok {
			return &r.schema.plans[idx], true
		}

		if idx, ok := r.schema.byName[strings.ToLower(key)]; ok {
			return &r.schema.plans[idx], true
		}

		return nil, false

	case func(*T) any:
		ptr := f(&r.value)

		return r.resolvePointerPlan(ptr)
	}

	return r.resolvePointerPlan(field)
}

func (r *Result[T]) resolvePointerPlan(fieldPtr any) (*fieldPlan, bool) {
	if fieldPtr == nil {
		return nil, false
	}

	rv := reflect.ValueOf(fieldPtr)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return nil, false
	}

	baseAddr := uintptr(unsafe.Pointer(&r.value))
	endAddr := baseAddr + reflect.TypeFor[T]().Size()
	ptrAddr := rv.Pointer()

	if ptrAddr < baseAddr || ptrAddr >= endAddr {
		return nil, false
	}

	offset := ptrAddr - baseAddr
	if idx, ok := r.schema.byOffset[offset]; ok {
		return &r.schema.plans[idx], true
	}

	return nil, false
}
