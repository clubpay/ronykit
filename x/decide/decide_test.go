package decide_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clubpay/ronykit/x/decide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Self-contained Enum & Score types (Option 3) ---

type Department string

const (
	DeptBilling Department = "billing"
	DeptSupport Department = "support"
	DeptSales   Department = "sales"
)

func (Department) Criteria() map[Department]string {
	return map[Department]string{
		DeptBilling: "Invoices, payments, refunds",
		DeptSupport: "Bugs, outages, technical issues",
		DeptSales:   "Pricing, new contracts",
	}
}

type Priority int

func (Priority) Levels() []string {
	return []string{
		"Low priority, can wait",
		"Medium priority, this week",
		"High priority, immediate blocker",
	}
}

func (Priority) MinLevel() int {
	return 1
}

type Channel string

func (Channel) Options() []Channel {
	return []Channel{"email", "slack", "phone"}
}

// --- Self-describing struct (Option 2) ---

type SupportTicket struct {
	Department Department
	Priority   Priority
	Channel    Channel
	Refund     bool
	ChurnRisk  float64
	Escalated  *bool
}

func (t *SupportTicket) Describe(b *decide.Builder[SupportTicket]) {
	b.Choice(&t.Department, "Which team should handle this ticket?")
	b.Score(&t.Priority, "How urgent is this ticket?")
	b.Choice(&t.Channel, "Preferred contact channel?")
	b.Bool(&t.Refund, "Does the customer explicitly request a refund?").
		Criteria("Asks for money back", "Does not ask for money back")
	b.Prob(&t.ChurnRisk, "The customer threatens to cancel their subscription.")
	b.Bool(&t.Escalated, "Requires executive escalation?").MinConfidence(0.85)
}

func TestSelfDescribingAndSelfContainedTypes(t *testing.T) {
	noulRefund := 0.92
	noulChurn := 0.81
	noulEscalated := 0.60 // answer confidence max(0.60, 0.40) = 0.60 < 0.85 -> stays nil
	choiceDept := "billing"
	choiceChan := "slack"
	scorePrio := 1.85
	confDept := 0.88
	confPrio := 0.79

	resolver := decide.NewStaticResolver(map[string]decide.Answer{
		"department": {
			Kind:   decide.KindChoice,
			Choice: &choiceDept,
			Probabilities: map[string]float64{
				"billing": 0.91,
				"support": 0.06,
				"sales":   0.03,
			},
			Confidence: &confDept,
		},
		"priority": {
			Kind:  decide.KindScore,
			Score: &scorePrio,
			Probabilities: map[string]float64{
				"0": 0.02,
				"1": 0.11,
				"2": 0.87,
			},
			Confidence: &confPrio,
		},
		"channel": {
			Kind:   decide.KindChoice,
			Choice: &choiceChan,
			Probabilities: map[string]float64{
				"email": 0.1,
				"slack": 0.8,
				"phone": 0.1,
			},
		},
		"refund": {
			Kind: decide.KindNoul,
			Noul: &noulRefund,
		},
		"churn_risk": {
			Kind: decide.KindNoul,
			Noul: &noulChurn,
		},
		"escalated": {
			Kind: decide.KindNoul,
			Noul: &noulEscalated,
		},
	})

	schema := decide.MustSchemaFor[SupportTicket](decide.WithResolver(resolver))

	res, err := schema.Decide(context.Background(), "Charged twice! Refund now or we cancel.")
	require.NoError(t, err)

	val := res.Value()
	assert.Equal(t, DeptBilling, val.Department)
	// Priority has MinLevel() = 1 and winning level index = 2 -> 1 + 2 = 3
	assert.Equal(t, Priority(3), val.Priority)
	assert.Equal(t, Channel("slack"), val.Channel)
	assert.True(t, val.Refund)
	assert.InDelta(t, 0.81, val.ChurnRisk, 1e-6)
	assert.Nil(t, val.Escalated, "Escalated should stay nil when confidence < 0.85")

	// Verify Result probability & confidence lookups by field pointer, selector, and name
	deptProbsByPtr := res.ProbabilitiesOf(&res.Ptr().Department)
	assert.InDelta(t, 0.91, deptProbsByPtr["billing"], 1e-6)

	deptProbsBySelector := res.ProbabilitiesOf(func(st *SupportTicket) any { return &st.Department })
	assert.Equal(t, deptProbsByPtr, deptProbsBySelector)

	deptProbsByName := res.ProbabilitiesOf("Department")
	assert.Equal(t, deptProbsByPtr, deptProbsByName)

	assert.InDelta(t, 0.88, res.ConfidenceOf(&res.Ptr().Department), 1e-6)
	assert.InDelta(t, 0.91, res.AnswerConfidenceOf(&res.Ptr().Department), 1e-6)
	assert.InDelta(t, 0.92, res.NoulOf(&res.Ptr().Refund), 1e-6)
	assert.InDelta(t, 1.85, res.ScoreOf(&res.Ptr().Priority), 1e-6)

	allProbs := res.Probabilities()
	require.Contains(t, allProbs, "department")
	require.Contains(t, allProbs, "refund")
	assert.InDelta(t, 0.92, allProbs["refund"]["true"], 1e-6)

	allConf := res.Confidence()
	assert.InDelta(t, 0.88, allConf["department"], 1e-6)
	assert.InDelta(t, 0.92, allConf["refund"], 1e-6)
}

func TestInlineBuilderAndNestedStructs(t *testing.T) {
	type BillingInfo struct {
		Tier      int
		RawScore  float64
		NeedsCall bool
	}

	type CustomerEval struct {
		Category string
		Billing  BillingInfo
	}

	schema := decide.MustSchema(func(b *decide.Builder[CustomerEval], c *CustomerEval) {
		b.Choice(&c.Category, "Customer segment").
			Option("enterprise", "Large organization").
			Option("smb", "Small business")

		b.Choice(&c.Billing.Tier, "Pricing tier").
			Options(10, 20, 30)

		b.Score(&c.Billing.RawScore, "Creditworthiness").
			Levels("Poor", "Fair", "Excellent").
			Min(1).
			Expected()

		b.Bool(&c.Billing.NeedsCall, "Requires account manager call?").
			ID("needs_am_call").
			Threshold(0.75)
	}, decide.WithResolver(decide.NewPassResolver()))

	questions := schema.Questions()
	require.Len(t, questions, 4)
	assert.Equal(t, "category", questions[0].ID)
	assert.Equal(t, "billing_tier", questions[1].ID)
	assert.Equal(t, "billing_raw_score", questions[2].ID)
	assert.Equal(t, "needs_am_call", questions[3].ID)

	res, err := schema.Decide(context.Background(), map[string]any{"company": "Acme Corp"})
	require.NoError(t, err)

	got := res.Value()
	assert.Equal(t, "enterprise", got.Category)
	assert.Equal(t, 10, got.Billing.Tier)
	assert.InDelta(t, 1.0, got.Billing.RawScore, 1e-6)
	assert.True(t, got.Billing.NeedsCall)
}

func TestJevResolverWireContract(t *testing.T) {
	type GateCard struct {
		Dept    string
		Urgency int
		Urgent  bool
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-jev-key", r.Header.Get("Authorization"))

		var reqBody map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		assert.Equal(t, "jev-latest", reqBody["model"])
		assert.Equal(t, "Stripe integration failing for 3 days", reqBody["state"])

		questions, ok := reqBody["questions"].(map[string]any)
		require.True(t, ok)
		require.Contains(t, questions, "dept")
		require.Contains(t, questions, "urgency")
		require.Contains(t, questions, "urgent")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "jev-1.13.0",
			"answers": {
				"dept": {
					"type": "choice",
					"choice": "technical",
					"probabilities": {"billing": 0.08, "technical": 0.85, "sales": 0.07},
					"confidence": 0.82
				},
				"urgency": {
					"type": "score",
					"score": 1.6,
					"legend": {"0": "Calm", "1": "Frustrated", "2": "Angry"},
					"probabilities": {"0": 0.05, "1": 0.30, "2": 0.65},
					"confidence": 0.78
				},
				"urgent": {
					"type": "noul",
					"noul": 0.92
				}
			},
			"usage": {"input_tokens": 312, "output_tokens": 0}
		}`))
	}))
	defer srv.Close()

	jev := decide.MustNewJevResolver(
		decide.WithJevEndpoint(srv.URL),
		decide.WithJevAPIKey("test-jev-key"),
	)

	schema := decide.MustSchema(func(b *decide.Builder[GateCard], g *GateCard) {
		b.Choice(&g.Dept, "Which team should handle this?").
			Option("billing", "Payment issues").
			Option("technical", "Bugs or integration problems").
			Option("sales", "")
		b.Score(&g.Urgency, "How frustrated the customer appears").
			Levels("Calm", "Frustrated", "Angry")
		b.Bool(&g.Urgent, "The message conveys urgency")
	}, decide.WithResolver(jev))

	res, err := schema.Decide(context.Background(), "Stripe integration failing for 3 days")
	require.NoError(t, err)

	assert.Equal(t, "jev-1.13.0", res.Model())
	assert.Equal(t, 312, res.Usage().InputTokens)
	assert.Equal(t, GateCard{
		Dept:    "technical",
		Urgency: 2,
		Urgent:  true,
	}, res.Value())
}

func TestLayaResolverAndBatch(t *testing.T) {
	type SentimentCard struct {
		Sentiment string
		IsSpam    bool
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/systemone", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model": "laya-rl-agent",
			"answers": {
				"sentiment": {
					"type": "choice",
					"choice": "positive",
					"probabilities": {"positive": 0.95, "negative": 0.05},
					"confidence": 0.81,
					"x_jev_confidence": 0.90,
					"action": {"act_probability": 1.0}
				},
				"is_spam": {
					"type": "noul",
					"noul": 0.04,
					"confidence": 0.96
				}
			},
			"usage": {
				"input_tokens": 64,
				"output_tokens": 0,
				"state_tokens": 12,
				"truncated": false
			},
			"routing": {
				"model": "english",
				"repo": "convaiinnovations/laya",
				"reason": "English Latin text"
			}
		}`))
	})

	mux.HandleFunc("/v1/systemone/batch", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		states, ok := req["states"].([]any)
		require.True(t, ok)
		require.Len(t, states, 2)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"results": [
				{
					"model": "laya",
					"answers": {
						"sentiment": {"type": "choice", "choice": "positive", "probabilities": {"positive": 0.9, "negative": 0.1}, "confidence": 0.8},
						"is_spam": {"type": "noul", "noul": 0.1}
					},
					"usage": {"input_tokens": 30, "output_tokens": 0}
				},
				{
					"model": "laya",
					"answers": {
						"sentiment": {"type": "choice", "choice": "negative", "probabilities": {"positive": 0.15, "negative": 0.85}, "confidence": 0.7},
						"is_spam": {"type": "noul", "noul": 0.88}
					},
					"usage": {"input_tokens": 34, "output_tokens": 0}
				}
			],
			"total_usage": {"input_tokens": 64, "output_tokens": 0}
		}`))
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	laya := decide.MustNewLayaResolver(
		decide.WithLayaEndpoint(srv.URL+"/v1/systemone"),
		decide.WithLayaModel("english"),
	)

	schema := decide.MustSchema(func(b *decide.Builder[SentimentCard], s *SentimentCard) {
		b.Choice(&s.Sentiment, "Overall sentiment").Options("positive", "negative")
		b.Bool(&s.IsSpam, "Message is spam")
	}, decide.WithResolver(laya), decide.WithLang("en"))

	singleRes, err := schema.Decide(context.Background(), "Loved it!")
	require.NoError(t, err)
	assert.Equal(t, SentimentCard{Sentiment: "positive", IsSpam: false}, singleRes.Value())
	require.NotNil(t, singleRes.Routing())
	assert.Equal(t, "english", singleRes.Routing().Model)

	ans, ok := singleRes.AnswerOf(&singleRes.Ptr().Sentiment)
	require.True(t, ok)
	require.NotNil(t, ans.JevConfidence)
	assert.InDelta(t, 0.90, *ans.JevConfidence, 1e-6)

	batchRes, err := schema.DecideBatch(context.Background(), []any{"Great product", "Buy cheap watches now"})
	require.NoError(t, err)
	require.Len(t, batchRes, 2)
	assert.Equal(t, SentimentCard{Sentiment: "positive", IsSpam: false}, batchRes[0].Value())
	assert.Equal(t, SentimentCard{Sentiment: "negative", IsSpam: true}, batchRes[1].Value())
}

func TestLLMResolverBothFormats(t *testing.T) {
	type ReviewGate struct {
		Scope   string
		Approve bool
		Quality int
	}

	t.Run("flat json from llm", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/chat/completions", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"model": "gpt-4o-mini",
				"choices": [{
					"message": {
						"content": "{\"scope\": \"small\", \"approve\": true, \"quality\": 2}"
					}
				}],
				"usage": {"prompt_tokens": 120, "completion_tokens": 18}
			}`))
		}))
		defer srv.Close()

		llm := decide.MustNewLLMResolver(
			decide.WithLLMBaseURL(srv.URL+"/v1"),
			decide.WithLLMAPIKey("sk-test"),
		)

		schema := decide.MustSchema(func(b *decide.Builder[ReviewGate], g *ReviewGate) {
			b.Choice(&g.Scope, "Diff scope").Options("small", "large")
			b.Bool(&g.Approve, "Ready to merge?")
			b.Score(&g.Quality, "Code quality").Levels("Poor", "OK", "Great")
		}, decide.WithResolver(llm))

		res, err := schema.Decide(context.Background(), "2 files changed")
		require.NoError(t, err)
		assert.Equal(t, ReviewGate{Scope: "small", Approve: true, Quality: 2}, res.Value())
		assert.Equal(t, 120, res.Usage().InputTokens)
		assert.Equal(t, 18, res.Usage().OutputTokens)
	})

	t.Run("structured answers with probabilities from llm", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"model": "gpt-4o-mini",
				"choices": [{
					"message": {
						"content": "~~~json\n{\"answers\": {\"scope\": {\"choice\": \"large\", \"probabilities\": {\"small\": 0.1, \"large\": 0.9}, \"confidence\": 0.85}, \"approve\": {\"noul\": 0.2, \"confidence\": 0.8}, \"quality\": {\"score\": 1, \"probabilities\": {\"0\": 0.1, \"1\": 0.8, \"2\": 0.1}, \"confidence\": 0.75}}}\n~~~"
					}
				}]
			}`))
		}))
		defer srv.Close()

		// Also verify markdown code fence stripping
		srvFence := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"model": "gpt-4o-mini",
				"choices": [{
					"message": {
						"content": "` + "```json\\n" + `{\"answers\": {\"scope\": {\"choice\": \"large\", \"probabilities\": {\"small\": 0.1, \"large\": 0.9}, \"confidence\": 0.85}, \"approve\": {\"noul\": 0.2, \"confidence\": 0.8}, \"quality\": {\"score\": 1, \"probabilities\": {\"0\": 0.1, \"1\": 0.8, \"2\": 0.1}, \"confidence\": 0.75}}}` + "\\n```" + `"
					}
				}]
			}`))
		}))
		defer srvFence.Close()

		llm := decide.MustNewLLMResolver(decide.WithLLMBaseURL(srvFence.URL))
		schema := decide.MustSchema(func(b *decide.Builder[ReviewGate], g *ReviewGate) {
			b.Choice(&g.Scope, "Diff scope").Options("small", "large")
			b.Bool(&g.Approve, "Ready to merge?")
			b.Score(&g.Quality, "Code quality").Levels("Poor", "OK", "Great")
		}, decide.WithResolver(llm))

		res, err := schema.Decide(context.Background(), "50 files changed")
		require.NoError(t, err)
		assert.Equal(t, ReviewGate{Scope: "large", Approve: false, Quality: 1}, res.Value())
		assert.InDelta(t, 0.9, res.ProbabilitiesOf(&res.Ptr().Scope)["large"], 1e-6)
	})
}

func TestValidationAndErrorClassification(t *testing.T) {
	type Simple struct {
		Flag   bool
		Choice string
		Score  int
	}

	t.Run("empty schema rejected", func(t *testing.T) {
		_, err := decide.NewSchema(func(_ *decide.Builder[Simple], _ *Simple) {})
		require.ErrorIs(t, err, decide.ErrInvalidSchema)
	})

	t.Run("choice without options rejected", func(t *testing.T) {
		_, err := decide.NewSchema(func(b *decide.Builder[Simple], s *Simple) {
			b.Choice(&s.Choice, "Pick one")
		})
		require.ErrorIs(t, err, decide.ErrInvalidQuestion)
	})

	t.Run("score with 1 level rejected", func(t *testing.T) {
		_, err := decide.NewSchema(func(b *decide.Builder[Simple], s *Simple) {
			b.Score(&s.Score, "Rate").Levels("Only one")
		})
		require.ErrorIs(t, err, decide.ErrInvalidQuestion)
	})

	t.Run("http 503 classified as ErrUnavailable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("overloaded"))
		}))
		defer srv.Close()

		jev := decide.MustNewJevResolver(
			decide.WithJevEndpoint(srv.URL),
			decide.WithJevAPIKey("key"),
		)
		schema := decide.MustSchema(func(b *decide.Builder[Simple], s *Simple) {
			b.Bool(&s.Flag, "Is valid?")
		}, decide.WithResolver(jev))

		_, err := schema.Decide(context.Background(), "test")
		require.ErrorIs(t, err, decide.ErrUnavailable)
	})

	t.Run("http 422 classified as ErrInvalidResponse", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"detail": "invalid model"}`))
		}))
		defer srv.Close()

		jev := decide.MustNewJevResolver(
			decide.WithJevEndpoint(srv.URL),
			decide.WithJevAPIKey("key"),
		)
		schema := decide.MustSchema(func(b *decide.Builder[Simple], s *Simple) {
			b.Bool(&s.Flag, "Is valid?")
		}, decide.WithResolver(jev))

		_, err := schema.Decide(context.Background(), "test")
		require.ErrorIs(t, err, decide.ErrInvalidResponse)
	})
}

func TestStateTemplateHelpers(t *testing.T) {
	type TicketInput struct {
		Customer string
		Tier     string
		Subject  string
		Body     string
		Tags     []string
	}

	type EvalCard struct {
		Urgent bool
	}

	const tmplStr = `Customer: {{ upper .Customer }} ({{ default "free" .Tier }})
Subject: {{ trim .Subject }}
Tags: {{ join .Tags ", " }}
Body: {{ .Body }}`

	tmpl := decide.MustTemplate[TicketInput](tmplStr, decide.WithTemplateName("ticket_state"))

	in := TicketInput{
		Customer: "Acme Corp",
		Subject:  "  Duplicate charge  ",
		Body:     "We were billed twice in March.",
		Tags:     []string{"billing", "march"},
	}

	rendered, err := tmpl.Render(in)
	require.NoError(t, err)
	assert.Equal(
		t,
		"Customer: ACME CORP (free)\nSubject: Duplicate charge\nTags: billing, march\nBody: We were billed twice in March.",
		rendered,
	)

	var capturedState any
	capturingResolver := decide.ResolverFunc(func(ctx context.Context, req decide.Request) (decide.Response, error) {
		capturedState = req.State

		return decide.NewPassResolver().Resolve(ctx, req)
	})

	schema := decide.MustSchema(func(b *decide.Builder[EvalCard], e *EvalCard) {
		b.Bool(&e.Urgent, "Is this urgent?")
	}, decide.WithResolver(capturingResolver))

	// 1. Pre-compiled StateTemplate.State(...)
	res, err := schema.Decide(context.Background(), tmpl.State(in))
	require.NoError(t, err)
	assert.True(t, res.Value().Urgent)
	assert.Equal(t, rendered, capturedState)

	// 2. One-shot decide.TemplateState(...)
	res, err = schema.Decide(context.Background(), decide.TemplateState(tmplStr, in))
	require.NoError(t, err)
	assert.True(t, res.Value().Urgent)
	assert.Equal(t, rendered, capturedState)

	// 3. Batch states via StateTemplate.States(...)
	batchRes, err := schema.DecideBatch(context.Background(), tmpl.States([]TicketInput{in, in}))
	require.NoError(t, err)
	require.Len(t, batchRes, 2)
}

