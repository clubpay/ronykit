package decide

// QuestionKind identifies one of the three System 1 question primitives.
type QuestionKind string

const (
	// KindNoul evaluates the probability in [0, 1] that a statement holds true.
	KindNoul QuestionKind = "noul"
	// KindChoice selects one option from a bounded set of labeled criteria.
	KindChoice QuestionKind = "choice"
	// KindScore rates a state across an ordered list of 2 to 10 rubric levels.
	KindScore QuestionKind = "score"
)

const (
	// DefaultNoulThreshold is the default probability threshold for hydrating a bool field.
	DefaultNoulThreshold = 0.5
	// MaxQuestions is the standard System 1 question limit per request.
	MaxQuestions = 32
	// MaxScoreLevels is the maximum number of levels in a score question.
	MaxScoreLevels = 10
	// MinScoreLevels is the minimum number of levels in a score question.
	MinScoreLevels = 2
	// MaxChoiceOptions is the maximum number of options in a choice question for Jev.
	MaxChoiceOptions = 255
	// LabelTrue is the standard probability key for a true KindNoul outcome.
	LabelTrue = "true"
	// LabelFalse is the standard probability key for a false KindNoul outcome.
	LabelFalse = "false"
)

// NoulCriteria optionally describes what true and false mean for a KindNoul question.
type NoulCriteria struct {
	True  string `json:"true,omitempty"`
	False string `json:"false,omitempty"`
}

// ChoiceOption represents a single labeled option and its optional rubric description.
type ChoiceOption struct {
	Label       string
	Description string
}

// Question defines a single typed question sent to a Resolver.
type Question struct {
	// ID is the unique key for this question within a Request.
	ID string
	// Kind is KindNoul, KindChoice, or KindScore.
	Kind QuestionKind
	// Instructions is the prompt or rubric question (string, map, or slice).
	Instructions any
	// Options holds the allowed choice options when Kind == KindChoice.
	Options []ChoiceOption
	// Levels holds the ordered level descriptions (low to high) when Kind == KindScore.
	Levels []string
	// NoulCriteria holds optional true/false descriptions when Kind == KindNoul.
	NoulCriteria *NoulCriteria
}

// OptionsMap returns the choice options as a label-to-description map.
func (q Question) OptionsMap() map[string]string {
	if len(q.Options) == 0 {
		return nil
	}

	out := make(map[string]string, len(q.Options))
	for _, opt := range q.Options {
		out[opt.Label] = opt.Description
	}

	return out
}

// Request is the low-level input to a Resolver.
type Request struct {
	// State is the input content to evaluate (string, map, struct, or slice).
	State any
	// Questions is the ordered list of questions to evaluate against State.
	Questions []Question
	// Model optionally overrides the resolver's default model.
	Model string
	// Task is an optional Laya routing task hint.
	Task string
	// Lang is an optional Laya language code hint.
	Lang string
	// MinConfidence is an optional Laya minimum confidence threshold.
	MinConfidence *float64
}

// BatchRequest evaluates the same Questions across multiple States.
type BatchRequest struct {
	// States is the list of input states to evaluate.
	States []any
	// Questions is the shared set of questions evaluated for each state.
	Questions []Question
	// Model optionally overrides the resolver's default model.
	Model string
	// Task is an optional Laya routing task hint.
	Task string
	// Lang is an optional Laya language code hint.
	Lang string
	// MinConfidence is an optional Laya minimum confidence threshold.
	MinConfidence *float64
}

// Answer holds the model's judgment and probability telemetry for a single Question.
type Answer struct {
	Kind          QuestionKind       `json:"type"`
	Noul          *float64           `json:"noul,omitempty"`
	Choice        *string            `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
	JevConfidence *float64           `json:"x_jev_confidence,omitempty"`
	Legend        map[string]string  `json:"legend,omitempty"`
	ActProb       *float64           `json:"act_probability,omitempty"`
}

// AnswerConfidence returns the probability mass on the reported answer:
// max(p) across Probabilities when present, or max(noul, 1-noul) for KindNoul.
// This quantity is scale-compatible across Jev, Laya, and LLM resolvers.
func (a Answer) AnswerConfidence() float64 {
	if len(a.Probabilities) > 0 {
		best := 0.0
		for _, p := range a.Probabilities {
			if p > best {
				best = p
			}
		}

		return best
	}

	if a.Noul != nil {
		p := *a.Noul
		if 1-p > p {
			return 1 - p
		}

		return p
	}

	if a.Confidence != nil {
		return *a.Confidence
	}

	return 0
}

// WinningLevelIndex returns the 0-based level index with the highest probability
// for a KindScore answer, or falls back to rounding Score.
func (a Answer) WinningLevelIndex(numLevels int) int {
	if len(a.Probabilities) > 0 {
		bestIdx := 0
		bestProb := -1.0

		for i := range numLevels {
			key := intToStr(i)
			if p, ok := a.Probabilities[key]; ok && p > bestProb {
				bestProb = p
				bestIdx = i
			}
		}

		if bestProb >= 0 {
			return bestIdx
		}
	}

	if a.Score != nil {
		idx := int(*a.Score + 0.5)
		if idx < 0 {
			return 0
		}

		if numLevels > 0 && idx >= numLevels {
			return numLevels - 1
		}

		return idx
	}

	return 0
}

// Usage reports token consumption and optional Laya truncation diagnostics.
type Usage struct {
	InputTokens        int      `json:"input_tokens"`
	OutputTokens       int      `json:"output_tokens"`
	StateTokens        int      `json:"state_tokens,omitempty"`
	StateTokensDropped int      `json:"state_tokens_dropped,omitempty"`
	Truncated          bool     `json:"truncated,omitempty"`
	TruncatedQuestions []string `json:"truncated_questions,omitempty"`
}

// RoutingInfo reports which Laya checkpoint handled the request and why.
type RoutingInfo struct {
	Model  string `json:"model,omitempty"`
	Repo   string `json:"repo,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// Response is the low-level output returned by a Resolver.
type Response struct {
	Model   string            `json:"model"`
	Answers map[string]Answer `json:"answers"`
	Usage   Usage             `json:"usage"`
	Routing *RoutingInfo      `json:"routing,omitempty"`
}

// BatchResponse is the output returned by a BatchResolver.
type BatchResponse struct {
	Results    []Response `json:"results"`
	TotalUsage Usage      `json:"total_usage"`
}
