// Example ex-14-decide demonstrates x/decide: schema-driven System 1 decision
// and structured extraction over Jev, Laya, and OpenAI-compatible LLMs, with
// template-based state generation and a rony HTTP service.
package main

import (
	"context"
	"os"
	"strings"

	"github.com/clubpay/ronykit/rony"
	"github.com/clubpay/ronykit/rony/errs"
	"github.com/clubpay/ronykit/x/decide"
	"github.com/clubpay/ronykit/x/rkit"
)

// --- 1. Self-Contained Enum & Rubric Types (Option 3) ---

type Department string

const (
	DeptBilling  Department = "billing"
	DeptPlatform Department = "platform"
	DeptSales    Department = "sales"
)

func (Department) Criteria() map[Department]string {
	return map[Department]string{
		DeptBilling:  "Invoices, duplicate charges, payment failures, or refund requests",
		DeptPlatform: "API errors, outages, webhook bugs, or integration issues",
		DeptSales:    "Enterprise pricing, seat expansion, or contract renewals",
	}
}

type Urgency int

func (Urgency) Levels() []string {
	return []string{
		"Low: general question, no deadline",
		"Medium: degraded workflow, needs attention this week",
		"High: production blocker, financial loss, or cancellation threat",
	}
}

func (Urgency) MinLevel() int {
	return 1
}

// --- 2. Self-Describing Decision Struct (Option 2) ---

type TriageDecision struct {
	Department Department `json:"department"`
	Urgency    Urgency    `json:"urgency"`
	Refund     bool       `json:"refund"`
	ChurnRisk  float64    `json:"churnRisk"`
}

func (t *TriageDecision) Describe(b *decide.Builder[TriageDecision]) {
	b.Choice(&t.Department, "Which team should own this ticket?")
	b.Score(&t.Urgency, "How urgent is this customer request?")
	b.Bool(&t.Refund, "Does the customer explicitly ask for a refund or credit?").
		Criteria("Asks for money back or credit", "No refund requested")
	b.Prob(&t.ChurnRisk, "The customer threatens to cancel or switch providers.")
}

// --- 3. Template-Based State Input ---

type TicketInput struct {
	Customer string   `json:"customer"`
	Plan     string   `json:"plan"`
	Subject  string   `json:"subject"`
	Message  string   `json:"message"`
	Tags     []string `json:"tags"`
}

var ticketStateTemplate = decide.MustTemplate[TicketInput](`
Customer: {{ upper .Customer }} (Plan: {{ default "starter" .Plan }})
Subject:  {{ trim .Subject }}
{{- if .Tags }}
Tags:     {{ join .Tags ", " }}
{{- end }}
---
{{ trim .Message }}
`)

// --- 4. Rony HTTP Service ---

type TriageService struct {
	schema *decide.Schema[TriageDecision]
}

func NewTriageService(resolver decide.Resolver) (*TriageService, error) {
	schema, err := decide.NewSchemaFor[TriageDecision](
		decide.WithResolver(resolver),
	)
	if err != nil {
		return nil, err
	}

	return &TriageService{schema: schema}, nil
}

type TriageResponse struct {
	RenderedState string                        `json:"renderedState"`
	Model         string                        `json:"model"`
	Decision      TriageDecision                `json:"decision"`
	Probabilities map[string]map[string]float64 `json:"probabilities"`
	Confidence    map[string]float64            `json:"confidence"`
}

type BatchTriageRequest struct {
	Tickets []TicketInput `json:"tickets"`
}

type BatchTriageResponse struct {
	Results []TriageResponse `json:"results"`
}

func (s *TriageService) Setup(srv *rony.Server) {
	rony.Setup(
		srv,
		"TriageService",
		rony.EmptyState(),
		rony.WithUnary(s.triage, rony.POST("/triage")),
		rony.WithUnary(s.triageBatch, rony.POST("/triage/batch")),
	)
}

func (s *TriageService) triage(ctx *rony.SUnaryCtx, req TicketInput) (*TriageResponse, error) {
	if strings.TrimSpace(req.Message) == "" {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("MESSAGE_REQUIRED").Err()
	}

	rendered, err := ticketStateTemplate.Render(req)
	if err != nil {
		return nil, errs.WrapCode(err, errs.InvalidArgument, "INVALID_STATE")
	}

	res, err := s.schema.Decide(ctx.Context(), ticketStateTemplate.State(req))
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "DECIDE_FAILED")
	}

	return &TriageResponse{
		RenderedState: rendered,
		Model:         res.Model(),
		Decision:      res.Value(),
		Probabilities: res.Probabilities(),
		Confidence:    res.Confidence(),
	}, nil
}

func (s *TriageService) triageBatch(
	ctx *rony.SUnaryCtx,
	req BatchTriageRequest,
) (*BatchTriageResponse, error) {
	if len(req.Tickets) == 0 {
		return &BatchTriageResponse{Results: []TriageResponse{}}, nil
	}

	batchRes, err := s.schema.DecideBatch(ctx.Context(), ticketStateTemplate.States(req.Tickets))
	if err != nil {
		return nil, errs.WrapCode(err, errs.Internal, "DECIDE_BATCH_FAILED")
	}

	out := make([]TriageResponse, len(batchRes))
	for i, res := range batchRes {
		out[i] = TriageResponse{
			RenderedState: ticketStateTemplate.MustRender(req.Tickets[i]),
			Model:         res.Model(),
			Decision:      res.Value(),
			Probabilities: res.Probabilities(),
			Confidence:    res.Confidence(),
		}
	}

	return &BatchTriageResponse{Results: out}, nil
}

// selectResolver picks Laya, Jev, or LLM from environment variables, falling back
// to an in-memory demo resolver so `go run .` works out of the box without API keys.
func selectResolver() decide.Resolver {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("DECIDE_PROVIDER"))) {
	case "jev":
		return decide.MustNewJevResolver()
	case "laya":
		return decide.MustNewLayaResolver()
	case "llm":
		return decide.MustNewLLMResolver()
	default:
		return newDemoKeywordResolver()
	}
}

// newDemoKeywordResolver simulates a System 1 model locally so the example can be
// explored without external API keys.
func newDemoKeywordResolver() decide.Resolver {
	return decide.ResolverFunc(func(_ context.Context, req decide.Request) (decide.Response, error) {
		text := rkit.TryCast[string](req.State)
		lower := strings.ToLower(text)

		dept := string(DeptPlatform)
		deptProbs := map[string]float64{
			string(DeptBilling):  0.10,
			string(DeptPlatform): 0.80,
			string(DeptSales):    0.10,
		}

		if strings.Contains(lower, "charge") || strings.Contains(lower, "invoice") ||
			strings.Contains(lower, "refund") {
			dept = string(DeptBilling)
			deptProbs = map[string]float64{
				string(DeptBilling):  0.92,
				string(DeptPlatform): 0.05,
				string(DeptSales):    0.03,
			}
		} else if strings.Contains(lower, "pricing") ||
			strings.Contains(lower, "contract") {
			dept = string(DeptSales)
			deptProbs = map[string]float64{
				string(DeptBilling):  0.05,
				string(DeptPlatform): 0.05,
				string(DeptSales):    0.90,
			}
		}

		refundNoul := 0.08
		if strings.Contains(lower, "refund") {
			refundNoul = 0.96
		}

		churnNoul := 0.12
		if strings.Contains(lower, "cancel") || strings.Contains(lower, "switch") {
			churnNoul = 0.89
		}

		urgencyScore := 0.2
		urgencyProbs := map[string]float64{"0": 0.80, "1": 0.15, "2": 0.05}

		if refundNoul > 0.5 || churnNoul > 0.5 || strings.Contains(lower, "outage") {
			urgencyScore = 1.85
			urgencyProbs = map[string]float64{"0": 0.02, "1": 0.11, "2": 0.87}
		}

		conf := 0.88

		return decide.Response{
			Model: "demo-local-v1",
			Answers: map[string]decide.Answer{
				"department": {
					Kind:          decide.KindChoice,
					Choice:        &dept,
					Probabilities: deptProbs,
					Confidence:    &conf,
				},
				"urgency": {
					Kind:          decide.KindScore,
					Score:         &urgencyScore,
					Probabilities: urgencyProbs,
					Confidence:    &conf,
				},
				"refund": {
					Kind:       decide.KindNoul,
					Noul:       &refundNoul,
					Confidence: &conf,
				},
				"churn_risk": {
					Kind:       decide.KindNoul,
					Noul:       &churnNoul,
					Confidence: &conf,
				},
			},
			Usage: decide.Usage{InputTokens: 64, OutputTokens: 0},
		}, nil
	})
}

func main() {
	ctx := context.Background()

	svc := rkit.Must(NewTriageService(selectResolver()))

	srv := rony.NewServer(
		rony.Listen(":8080"),
		rony.WithServerName("TriageServer"),
		rony.WithAPIDocs("/docs"),
		rony.UseScalarUI(),
	)
	svc.Setup(srv)

	rkit.Assert(srv.Run(ctx, os.Kill, os.Interrupt))
}
