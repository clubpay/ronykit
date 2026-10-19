package decide

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/clubpay/ronykit/x/rkit"
)

const (
	// DefaultLayaEndpoint is Laya Studio's hosted `/v1/systemone` endpoint.
	// Self-hosted `laya-serve` instances typically listen on `http://127.0.0.1:8000/v1/systemone`.
	DefaultLayaEndpoint = "https://api.laya.studio/v1/systemone"
	// MaxLayaBatchStates is the maximum number of states per `/v1/systemone/batch` call.
	MaxLayaBatchStates = 64
)

type layaConfig struct {
	endpoint      string
	batchEndpoint string
	apiKey        string
	model         string
	httpClient    *http.Client
}

// LayaOption configures a LayaResolver.
type LayaOption func(*layaConfig)

// WithLayaEndpoint overrides the Laya `/v1/systemone` endpoint URL.
func WithLayaEndpoint(endpoint string) LayaOption {
	return func(c *layaConfig) {
		c.endpoint = strings.TrimSpace(endpoint)
	}
}

// WithLayaBatchEndpoint overrides the Laya `/v1/systemone/batch` endpoint URL.
// When unset, `/batch` is appended to the configured `/v1/systemone` endpoint.
func WithLayaBatchEndpoint(batchEndpoint string) LayaOption {
	return func(c *layaConfig) {
		c.batchEndpoint = strings.TrimSpace(batchEndpoint)
	}
}

// WithLayaAPIKey sets the optional Bearer token for Laya Studio or protected `laya-serve` instances.
func WithLayaAPIKey(apiKey string) LayaOption {
	return func(c *layaConfig) {
		c.apiKey = strings.TrimSpace(apiKey)
	}
}

// WithLayaModel sets an optional default Laya checkpoint name (e.g. "english", "multilingual",
// "typed-decisions", or "laya"). When empty, Laya's router selects the checkpoint automatically.
func WithLayaModel(model string) LayaOption {
	return func(c *layaConfig) {
		c.model = strings.TrimSpace(model)
	}
}

// WithLayaHTTPClient sets a custom *http.Client for Laya API calls.
func WithLayaHTTPClient(client *http.Client) LayaOption {
	return func(c *layaConfig) {
		c.httpClient = client
	}
}

// LayaResolver evaluates questions against Laya (`laya-serve` or Laya Studio) over
// `/v1/systemone` and `/v1/systemone/batch`.
type LayaResolver struct {
	cfg layaConfig
}

// NewLayaResolver creates a new LayaResolver.
// Unset configuration fields fall back to environment variables:
//   - DECIDE_LAYA_ENDPOINT (defaults to DefaultLayaEndpoint)
//   - DECIDE_LAYA_API_KEY (or LAYA_API_KEY; optional for self-hosted laya-serve)
//   - DECIDE_LAYA_MODEL (optional)
func NewLayaResolver(opts ...LayaOption) (*LayaResolver, error) {
	cfg := layaConfig{
		endpoint: strings.TrimSpace(os.Getenv("DECIDE_LAYA_ENDPOINT")),
		apiKey: rkit.Coalesce(
			strings.TrimSpace(os.Getenv("DECIDE_LAYA_API_KEY")),
			strings.TrimSpace(os.Getenv("LAYA_API_KEY")),
		),
		model:      strings.TrimSpace(os.Getenv("DECIDE_LAYA_MODEL")),
		httpClient: defaultHTTPClient,
	}

	for _, o := range opts {
		o(&cfg)
	}

	cfg.endpoint = rkit.Coalesce(cfg.endpoint, DefaultLayaEndpoint)
	if cfg.batchEndpoint == "" {
		cfg.batchEndpoint = strings.TrimRight(cfg.endpoint, "/") + "/batch"
	}

	cfg.httpClient = rkit.Coalesce(cfg.httpClient, defaultHTTPClient)

	return &LayaResolver{cfg: cfg}, nil
}

// MustNewLayaResolver is like NewLayaResolver but panics on error.
func MustNewLayaResolver(opts ...LayaOption) *LayaResolver {
	return rkit.Must(NewLayaResolver(opts...))
}

// Resolve implements Resolver.
func (r *LayaResolver) Resolve(ctx context.Context, req Request) (Response, error) {
	model := rkit.Coalesce(strings.TrimSpace(req.Model), r.cfg.model)

	rawReq, err := buildSystemOneRequest(req, model, true)
	if err != nil {
		return Response{}, err
	}

	payload, err := doSystemOnePost(ctx, r.cfg.httpClient, r.cfg.endpoint, r.cfg.apiKey, rawReq)
	if err != nil {
		return Response{}, err
	}

	return parseSystemOneResponse(req.Questions, payload)
}

// ResolveBatch implements BatchResolver using Laya's `/v1/systemone/batch` endpoint.
// If more than MaxLayaBatchStates (64) states are provided, ResolveBatch transparently
// paginates them in chunks of 64.
func (r *LayaResolver) ResolveBatch(ctx context.Context, req BatchRequest) (BatchResponse, error) {
	if len(req.States) == 0 {
		return BatchResponse{}, nil
	}

	allResults := make([]Response, 0, len(req.States))

	var totalUsage Usage

	err := rkit.Paginate(req.States, MaxLayaBatchStates, func(start, end int) error {
		chunkReq := req
		chunkReq.States = req.States[start:end]

		chunkResp, err := r.resolveBatchChunk(ctx, chunkReq)
		if err != nil {
			return err
		}

		allResults = append(allResults, chunkResp.Results...)
		totalUsage.InputTokens += chunkResp.TotalUsage.InputTokens
		totalUsage.OutputTokens += chunkResp.TotalUsage.OutputTokens
		totalUsage.StateTokens += chunkResp.TotalUsage.StateTokens
		totalUsage.StateTokensDropped += chunkResp.TotalUsage.StateTokensDropped
		totalUsage.Truncated = totalUsage.Truncated || chunkResp.TotalUsage.Truncated

		return nil
	})
	if err != nil {
		return BatchResponse{}, err
	}

	return BatchResponse{
		Results:    allResults,
		TotalUsage: totalUsage,
	}, nil
}

func (r *LayaResolver) resolveBatchChunk(ctx context.Context, req BatchRequest) (BatchResponse, error) {
	model := rkit.Coalesce(strings.TrimSpace(req.Model), r.cfg.model)

	rawReq, err := buildSystemOneBatchRequest(req, model)
	if err != nil {
		return BatchResponse{}, err
	}

	payload, err := doSystemOnePost(ctx, r.cfg.httpClient, r.cfg.batchEndpoint, r.cfg.apiKey, rawReq)
	if err != nil {
		return BatchResponse{}, err
	}

	var wire systemOneBatchResponseWire
	if err := json.Unmarshal(payload, &wire); err != nil {
		return BatchResponse{}, fmt.Errorf("%w: decode batch response: %w", ErrInvalidResponse, err)
	}

	if len(wire.Results) != len(req.States) {
		return BatchResponse{}, fmt.Errorf(
			"%w: batch expected %d results, got %d",
			ErrInvalidResponse, len(req.States), len(wire.Results),
		)
	}

	results := make([]Response, len(wire.Results))
	for i, item := range wire.Results {
		decoded, err := decodeSystemOneWire(req.Questions, item)
		if err != nil {
			return BatchResponse{}, fmt.Errorf("batch item %d: %w", i, err)
		}

		results[i] = decoded
	}

	return BatchResponse{
		Results:    results,
		TotalUsage: wire.TotalUsage,
	}, nil
}
