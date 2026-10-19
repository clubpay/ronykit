package decide

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/clubpay/ronykit/x/rkit"
)

const (
	// DefaultJevEndpoint is TypeSafe Jev's hosted System 1 endpoint.
	DefaultJevEndpoint = "https://api.typesafe.ai/v1/systemone"
	// DefaultJevModel is the default Jev model alias.
	DefaultJevModel = "jev-latest"
	// DefaultHTTPTimeout is the default HTTP client timeout for resolvers.
	DefaultHTTPTimeout = 2 * time.Minute
)

var defaultHTTPClient = &http.Client{Timeout: DefaultHTTPTimeout}

type jevConfig struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
}

// JevOption configures a JevResolver.
type JevOption func(*jevConfig)

// WithJevEndpoint overrides the Jev `/v1/systemone` endpoint URL.
func WithJevEndpoint(endpoint string) JevOption {
	return func(c *jevConfig) {
		c.endpoint = strings.TrimSpace(endpoint)
	}
}

// WithJevAPIKey sets the Bearer token for Jev API calls.
func WithJevAPIKey(apiKey string) JevOption {
	return func(c *jevConfig) {
		c.apiKey = strings.TrimSpace(apiKey)
	}
}

// WithJevModel sets the default Jev model (defaults to "jev-latest").
func WithJevModel(model string) JevOption {
	return func(c *jevConfig) {
		c.model = strings.TrimSpace(model)
	}
}

// WithJevHTTPClient sets a custom *http.Client for Jev API calls.
func WithJevHTTPClient(client *http.Client) JevOption {
	return func(c *jevConfig) {
		c.httpClient = client
	}
}

// JevResolver evaluates questions against TypeSafe Jev's `/v1/systemone` HTTP API.
type JevResolver struct {
	cfg jevConfig
}

// NewJevResolver creates a new JevResolver.
// Unset configuration fields fall back to environment variables:
//   - DECIDE_JEV_API_KEY (or TYPESAFE_API_KEY)
//   - DECIDE_JEV_ENDPOINT (defaults to DefaultJevEndpoint)
//   - DECIDE_JEV_MODEL (defaults to DefaultJevModel)
func NewJevResolver(opts ...JevOption) (*JevResolver, error) {
	cfg := jevConfig{
		endpoint: strings.TrimSpace(os.Getenv("DECIDE_JEV_ENDPOINT")),
		apiKey: rkit.Coalesce(
			strings.TrimSpace(os.Getenv("DECIDE_JEV_API_KEY")),
			strings.TrimSpace(os.Getenv("TYPESAFE_API_KEY")),
		),
		model:      strings.TrimSpace(os.Getenv("DECIDE_JEV_MODEL")),
		httpClient: defaultHTTPClient,
	}

	for _, o := range opts {
		o(&cfg)
	}

	cfg.endpoint = rkit.Coalesce(cfg.endpoint, DefaultJevEndpoint)
	cfg.model = rkit.Coalesce(cfg.model, DefaultJevModel)

	cfg.httpClient = rkit.Coalesce(cfg.httpClient, defaultHTTPClient)

	if cfg.apiKey == "" {
		return nil, fmt.Errorf("%w: jev api key is not configured", ErrInvalidSchema)
	}

	return &JevResolver{cfg: cfg}, nil
}

// MustNewJevResolver is like NewJevResolver but panics on error.
func MustNewJevResolver(opts ...JevOption) *JevResolver {
	return rkit.Must(NewJevResolver(opts...))
}

// Resolve implements Resolver.
func (r *JevResolver) Resolve(ctx context.Context, req Request) (Response, error) {
	model := rkit.Coalesce(strings.TrimSpace(req.Model), r.cfg.model, DefaultJevModel)

	rawReq, err := buildSystemOneRequest(req, model, false)
	if err != nil {
		return Response{}, err
	}

	payload, err := doSystemOnePost(ctx, r.cfg.httpClient, r.cfg.endpoint, r.cfg.apiKey, rawReq)
	if err != nil {
		return Response{}, err
	}

	return parseSystemOneResponse(req.Questions, payload)
}

type systemOneRequestWire struct {
	State         any                              `json:"state"`
	Model         string                           `json:"model,omitempty"`
	Questions     map[string]systemOneQuestionWire `json:"questions"`
	Task          string                           `json:"task,omitempty"`
	Lang          string                           `json:"lang,omitempty"`
	MinConfidence *float64                         `json:"min_confidence,omitempty"`
}

type systemOneBatchRequestWire struct {
	States        []any                            `json:"states"`
	Model         string                           `json:"model,omitempty"`
	Questions     map[string]systemOneQuestionWire `json:"questions"`
	Task          string                           `json:"task,omitempty"`
	Lang          string                           `json:"lang,omitempty"`
	MinConfidence *float64                         `json:"min_confidence,omitempty"`
}

type systemOneQuestionWire struct {
	Type         QuestionKind `json:"type"`
	Instructions any          `json:"instructions,omitempty"`
	Criteria     any          `json:"criteria,omitempty"`
}

type systemOneAnswerWire struct {
	Type          string             `json:"type"`
	Noul          any                `json:"noul"`
	Choice        any                `json:"choice"`
	Score         any                `json:"score"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    any                `json:"confidence"`
	JevConfidence any                `json:"x_jev_confidence"`
	Legend        map[string]string  `json:"legend"`
	Action        *struct {
		ActProbability *float64 `json:"act_probability"`
	} `json:"action"`
}

type systemOneResponseWire struct {
	Model   string                         `json:"model"`
	Answers map[string]systemOneAnswerWire `json:"answers"`
	Usage   Usage                          `json:"usage"`
	Routing *RoutingInfo                   `json:"routing"`
}

type systemOneBatchResponseWire struct {
	Results    []systemOneResponseWire `json:"results"`
	TotalUsage Usage                   `json:"total_usage"`
}

func buildSystemOneQuestions(questions []Question) (map[string]systemOneQuestionWire, error) {
	if len(questions) == 0 {
		return nil, fmt.Errorf("%w: at least one question is required", ErrInvalidQuestion)
	}

	encoded := make(map[string]systemOneQuestionWire, len(questions))

	for _, q := range questions {
		if err := ValidateQuestion(q); err != nil {
			return nil, err
		}

		instr := q.Instructions
		if instr == nil || instr == "" {
			instr = strings.ReplaceAll(q.ID, "_", " ")
		}

		wire := systemOneQuestionWire{
			Type:         q.Kind,
			Instructions: instr,
		}

		switch q.Kind {
		case KindNoul:
			if q.NoulCriteria != nil && (q.NoulCriteria.True != "" || q.NoulCriteria.False != "") {
				wire.Criteria = q.NoulCriteria
			}
		case KindChoice:
			criteria := make(map[string]*string, len(q.Options))
			for _, opt := range q.Options {
				if opt.Description == "" {
					criteria[opt.Label] = nil
				} else {
					desc := opt.Description
					criteria[opt.Label] = &desc
				}
			}

			wire.Criteria = criteria
		case KindScore:
			wire.Criteria = q.Levels
		}

		encoded[q.ID] = wire
	}

	return encoded, nil
}

func buildSystemOneRequest(req Request, model string, includeLayaFields bool) ([]byte, error) {
	questions, err := buildSystemOneQuestions(req.Questions)
	if err != nil {
		return nil, err
	}

	resolvedState, err := resolveStateInput(req.State)
	if err != nil {
		return nil, err
	}

	if resolvedState == nil {
		resolvedState = ""
	}

	wire := systemOneRequestWire{
		State:     resolvedState,
		Model:     model,
		Questions: questions,
	}

	if includeLayaFields {
		wire.Task = req.Task
		wire.Lang = req.Lang
		wire.MinConfidence = req.MinConfidence
	}

	return json.Marshal(wire)
}

func buildSystemOneBatchRequest(req BatchRequest, model string) ([]byte, error) {
	if len(req.States) == 0 {
		return nil, fmt.Errorf("%w: batch states must not be empty", ErrInvalidQuestion)
	}

	resolvedStates, err := resolveBatchStatesInput(req.States)
	if err != nil {
		return nil, err
	}

	questions, err := buildSystemOneQuestions(req.Questions)
	if err != nil {
		return nil, err
	}

	wire := systemOneBatchRequestWire{
		States:        resolvedStates,
		Model:         model,
		Questions:     questions,
		Task:          req.Task,
		Lang:          req.Lang,
		MinConfidence: req.MinConfidence,
	}

	return json.Marshal(wire)
}

func doSystemOnePost(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	apiKey string,
	rawBody []byte,
) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(rawBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %w", ErrUnavailable, err)
	}

	if err := classifyHTTPStatus(resp.StatusCode, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

func classifyHTTPStatus(status int, payload []byte) error {
	if status == http.StatusTooManyRequests || status >= http.StatusInternalServerError {
		return fmt.Errorf("%w: http %d: %s", ErrUnavailable, status, strings.TrimSpace(rkit.B2S(payload)))
	}

	if status >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"%w: http %d: %s",
			ErrInvalidResponse, status, strings.TrimSpace(rkit.B2S(payload)),
		)
	}

	return nil
}

func parseSystemOneResponse(questions []Question, body []byte) (Response, error) {
	var wire systemOneResponseWire
	if err := json.Unmarshal(body, &wire); err != nil {
		return Response{}, fmt.Errorf("%w: decode response: %w", ErrInvalidResponse, err)
	}

	return decodeSystemOneWire(questions, wire)
}

func decodeSystemOneWire(questions []Question, wire systemOneResponseWire) (Response, error) {
	answers := make(map[string]Answer, len(questions))

	for _, q := range questions {
		rawAns, ok := wire.Answers[q.ID]
		if !ok {
			return Response{}, fmt.Errorf("%w: answer missing for question %q", ErrInvalidResponse, q.ID)
		}

		ans := Answer{
			Kind:          q.Kind,
			Probabilities: rawAns.Probabilities,
			Legend:        rawAns.Legend,
		}

		if c, ok := parseFloatAny(rawAns.Confidence); ok {
			ans.Confidence = &c
		}

		if jc, ok := parseFloatAny(rawAns.JevConfidence); ok {
			ans.JevConfidence = &jc
		}

		if rawAns.Action != nil && rawAns.Action.ActProbability != nil {
			ap := *rawAns.Action.ActProbability
			ans.ActProb = &ap
		}

		switch q.Kind {
		case KindNoul:
			noul, ok := parseFloatAny(rawAns.Noul)
			if !ok {
				return Response{}, fmt.Errorf(
					"%w: noul for question %q is not a number",
					ErrInvalidResponse, q.ID,
				)
			}

			ans.Noul = &noul

		case KindChoice:
			choice, ok := parseChoiceAny(rawAns.Choice)
			if !ok || choice == "" {
				return Response{}, fmt.Errorf(
					"%w: choice for question %q is missing",
					ErrInvalidResponse, q.ID,
				)
			}

			ans.Choice = &choice

		case KindScore:
			score, ok := parseFloatAny(rawAns.Score)
			if !ok {
				return Response{}, fmt.Errorf(
					"%w: score for question %q is not a number",
					ErrInvalidResponse, q.ID,
				)
			}

			ans.Score = &score

		default:
			return Response{}, fmt.Errorf("%w: unknown question kind %q", ErrInvalidQuestion, q.Kind)
		}

		answers[q.ID] = ans
	}

	return Response{
		Model:   wire.Model,
		Answers: answers,
		Usage:   wire.Usage,
		Routing: wire.Routing,
	}, nil
}

func parseFloatAny(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()

		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)

		return f, err == nil
	default:
		return 0, false
	}
}

func parseChoiceAny(v any) (string, bool) {
	switch s := v.(type) {
	case string:
		return s, true
	case float64:
		return strconv.FormatFloat(s, 'f', -1, 64), true //rkitcheck:ignore float choice label
	case int:
		return rkit.IntToStr(s), true
	case int64:
		return rkit.Int64ToStr(s), true
	case json.Number:
		return s.String(), true
	default:
		return "", false
	}
}
