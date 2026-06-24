package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clubpay/ronykit/intent"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/rony"
	"github.com/clubpay/ronykit/rony/errs"
	staticknowledge "github.com/clubpay/ronykit/std/knowledge/static"
	inmemmemory "github.com/clubpay/ronykit/std/memories/inmem"
	"github.com/clubpay/ronykit/x/telemetry/logkit"
)

func main() {
	ctx := context.Background()

	knowledgeDir := envOr("KNOWLEDGE_DIR", "knowledge")
	knowledgeStore, err := staticknowledge.LoadDir(knowledgeDir)
	if err != nil {
		panic(fmt.Errorf("load knowledge from %q: %w", knowledgeDir, err))
	}

	mem := inmemmemory.New()
	sessions := intent.NewSessionManager(mem)

	tools, err := newToolRegistry()
	if err != nil {
		panic(err)
	}

	pool, err := newLLMPool()
	if err != nil {
		panic(err)
	}

	state := &agentState{models: modelSummary(pool)}

	agent := intent.New(
		intent.WithName("demo-agent"),
		intent.WithStaticKnowledge(knowledgeStore),
		intent.WithSkills(knowledgeStore.Skills()),
		intent.WithLLMPool(pool),
		intent.WithMemory(mem),
		intent.WithTools(tools),
		intent.WithSessions(sessions),
		intent.WithLogger(logkit.New(logkit.WithJSON(), logkit.WithLevel(logkit.DebugLevel)).SLog()),
		intent.WithService(intent.ServiceDescriptor{
			Name: "chat",
			Mount: func(m intent.EndpointMount) error {
				intent.Setup(
					m,
					"AgentService",
					rony.ToInitiateState(state),
					rony.WithMiddleware[*agentState](baseHdrMW),
					rony.WithUnary(
						chat,
						rony.POST("/chat"),
					),
					rony.WithUnary(
						createSession,
						rony.POST("/session", rony.UnaryName("create-session")),
					),
					rony.WithUnary(
						getHistory,
						rony.GET("/session/{sessionId}/history"),
					),
				)

				return nil
			},
		}),
		intent.WithServerOption(
			rony.Listen(envOr("LISTEN_ADDR", ":8082")),
			rony.WithServerName("IntentAgent"),
			rony.WithAPIDocs("/docs"),
		),
	)
	state.agent = agent

	fmt.Printf("intent agent example (models=%s, knowledge=%s)\n",
		state.models,
		filepath.Clean(knowledgeDir),
	)

	err = agent.Run(ctx, os.Interrupt, os.Kill)
	if err != nil {
		panic(err)
	}
}

var baseHDR = map[string]string{
	"Content-Type": "application/json",
}

func baseHdrMW(ctx *kit.Context) {
	ctx.PresetHdrMap(baseHDR)
}

type agentState struct {
	models string
	agent  *intent.Agent
}

func (s *agentState) Name() string { return "AgentState" }

func (s *agentState) Reduce(_ string) error { return nil }

type ChatRequest struct {
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
}

type ChatResponse struct {
	SessionID string `json:"sessionId"`
	Reply     string `json:"reply"`
	Provider  string `json:"provider"`
}

type CreateSessionRequest struct {
	SessionID string            `json:"sessionId"`
	Metadata  map[string]string `json:"metadata"`
}

type CreateSessionResponse struct {
	SessionID string `json:"sessionId"`
}

type HistoryRequest struct {
	SessionID string `json:"sessionId"`
}

type HistoryResponse struct {
	SessionID string       `json:"sessionId"`
	Messages  []llmMessage `json:"messages"`
}

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RContext = rony.UnaryCtx[*agentState, string]

func chat(ctx *RContext, req ChatRequest) (*ChatResponse, error) {
	if req.Message == "" {
		return nil, errs.WrapCode(fmt.Errorf("message is required"), errs.InvalidArgument, "BAD_REQUEST")
	}

	agent := ctx.State().agent
	if agent == nil {
		return nil, errs.B().Code(errs.Internal).Msg("agent is not initialized").Err()
	}

	sess, err := resolveSession(ctx.Context(), agent.Sessions(), req.SessionID)
	if err != nil {
		return nil, errs.B().Msg(err.Error()).Err()
	}

	result, err := agent.RunTurn(ctx.Context(), intent.TurnInput{
		Session:     sess,
		UserMessage: intent.Message{Role: intent.RoleHuman, Parts: []intent.Part{intent.TextPart(req.Message)}},
	})
	if err != nil {
		return nil, errs.B().Code(errs.Internal).Cause(err).Msg(err.Error()).Err()
	}

	return &ChatResponse{
		SessionID: sess.ID,
		Reply:     result.Response.Content,
		Provider:  ctx.State().models,
	}, nil
}

func createSession(ctx *RContext, req CreateSessionRequest) (*CreateSessionResponse, error) {
	agent := ctx.State().agent
	if agent == nil {
		return nil, errs.B().Code(errs.Internal).Msg("agent is not initialized").Err()
	}

	var opts []intent.SessionOption
	if req.SessionID != "" {
		opts = append(opts, intent.SessionWithID(req.SessionID))
	}

	if len(req.Metadata) > 0 {
		opts = append(opts, intent.SessionWithMetadata(req.Metadata))
	}

	sess, err := agent.Sessions().Create(ctx.Context(), opts...)
	if err != nil {
		return nil, err
	}

	return &CreateSessionResponse{SessionID: sess.ID}, nil
}

func getHistory(ctx *RContext, req HistoryRequest) (*HistoryResponse, error) {
	if req.SessionID == "" {
		return nil, errs.WrapCode(fmt.Errorf("sessionId is required"), errs.InvalidArgument, "BAD_REQUEST")
	}

	agent := ctx.State().agent
	if agent == nil {
		return nil, errs.B().Code(errs.Internal).Msg("agent is not initialized").Err()
	}

	sess, err := agent.Sessions().Get(ctx.Context(), req.SessionID)
	if err != nil {
		return nil, err
	}

	history, err := intent.LoadHistory(ctx.Context(), sess)
	if err != nil {
		return nil, err
	}

	messages := make([]llmMessage, 0, len(history))
	for _, msg := range history {
		messages = append(messages, llmMessage{
			Role:    string(msg.Role),
			Content: messageText(msg),
		})
	}

	return &HistoryResponse{
		SessionID: sess.ID,
		Messages:  messages,
	}, nil
}

func resolveSession(ctx context.Context, mgr *intent.SessionManager, sessionID string) (*intent.Session, error) {
	if sessionID != "" {
		return mgr.Get(ctx, sessionID)
	}

	return mgr.Create(ctx)
}

func messageText(msg intent.Message) string {
	var parts []string
	for _, part := range msg.Parts {
		if part.Text != "" {
			parts = append(parts, part.Text)
		}
	}

	return strings.Join(parts, "\n")
}
