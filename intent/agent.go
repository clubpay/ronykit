package intent

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/clubpay/ronykit/intent/errs"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/rony"
)

// Agent wraps a rony.Server with agent capabilities: knowledge, LLMs, memory,
// MCP servers, and rony endpoints.
// Agent is the public entry point for running turns and serving endpoints.
type Agent struct {
	cfg agentConfig
	srv *rony.Server
	rt  *runtime
}

// Config exposes the agent's composed dependencies.
type Config struct {
	Name            string
	Knowledge       Knowledge
	StaticKnowledge StaticStore
	Retriever       Retriever
	LLM             Pool
	Memory          Memory
	MCPServers      MCPRegistry
	Tools           ToolRegistry
	Skills          SkillRegistry
	Sessions        *SessionManager
	Services        ServiceRegistry
	Executor        TaskExecutor
}

type agentConfig struct {
	Config
	serverOpts        []rony.ServerOption
	externalSrv       *rony.Server
	services          []ServiceDescriptor
	maxToolIterations int
	logger            *slog.Logger
}

func defaultAgentConfig() agentConfig {
	return agentConfig{}
}

// New builds an Agent and mounts service descriptors on the underlying rony.Server.
func New(opts ...Option) *Agent {
	cfg := defaultAgentConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	rt := newRuntime(cfg)

	var srv *rony.Server
	if cfg.externalSrv != nil {
		srv = cfg.externalSrv
	} else {
		srv = rony.NewServer(cfg.serverOpts...)
	}

	agent := &Agent{
		cfg: cfg,
		srv: srv,
		rt:  rt,
	}

	mount := EndpointMount{srv: srv}
	agent.mountServices(mount)

	return agent
}

func (a *Agent) mountServices(mount EndpointMount) {
	if a.cfg.Services != nil {
		for _, d := range a.cfg.Services.All() {
			a.mountService(mount, d)
		}
	} else {
		for _, d := range a.cfg.services {
			a.mountService(mount, d)
		}
	}
}

func (a *Agent) mountService(mount EndpointMount, desc ServiceDescriptor) {
	if desc.Mount == nil {
		return
	}

	err := desc.Mount(mount)
	if err != nil {
		panic(err)
	}
}

// Config returns a snapshot of the agent configuration.
func (a *Agent) Config() Config {
	return a.cfg.Config
}

// Sessions returns the configured session manager, if any.
func (a *Agent) Sessions() *SessionManager {
	return a.cfg.Sessions
}

// RunTurn executes one agent loop turn for a session message.
func (a *Agent) RunTurn(ctx context.Context, in TurnInput) (TurnResult, error) {
	if a.rt == nil {
		return TurnResult{}, errs.Wrap(errs.ErrUnsupportedOperation, "agent llm pool is required")
	}

	return a.rt.runTurn(ctx, in)
}

// Server returns the underlying rony.Server.
func (a *Agent) Server() *rony.Server {
	return a.srv
}

// Start starts the underlying rony.Server.
func (a *Agent) Start(ctx context.Context) error {
	return a.srv.Start(ctx)
}

// Stop shuts down the underlying rony.Server.
func (a *Agent) Stop(ctx context.Context, signals ...os.Signal) {
	a.srv.Stop(ctx, signals...)
}

// Run starts the agent in blocking mode.
func (a *Agent) Run(ctx context.Context, signals ...os.Signal) error {
	return a.srv.Run(ctx, signals...)
}

// PrintRoutes writes registered routes to w.
func (a *Agent) PrintRoutes(w io.Writer) {
	a.srv.PrintRoutes(w)
}

// ExportDesc returns service descriptions from the underlying server.
func (a *Agent) ExportDesc() []desc.ServiceDesc {
	return a.srv.ExportDesc()
}
