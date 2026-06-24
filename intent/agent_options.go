package intent

import (
	"log/slog"

	"github.com/clubpay/ronykit/rony"
)

// Option configures an Agent.
type Option func(cfg *agentConfig)

func WithName(name string) Option {
	return func(cfg *agentConfig) {
		cfg.Name = name
	}
}

func WithKnowledge(k Knowledge) Option {
	return func(cfg *agentConfig) {
		cfg.Knowledge = k
	}
}

func WithLLMPool(pool Pool) Option {
	return func(cfg *agentConfig) {
		cfg.LLM = pool
	}
}

func WithMemory(m Memory) Option {
	return func(cfg *agentConfig) {
		cfg.Memory = m
	}
}

func WithMCPServers(reg MCPRegistry) Option {
	return func(cfg *agentConfig) {
		cfg.MCPServers = reg
	}
}

func WithServiceRegistry(reg ServiceRegistry) Option {
	return func(cfg *agentConfig) {
		cfg.Services = reg
	}
}

func WithService(desc ServiceDescriptor) Option {
	return func(cfg *agentConfig) {
		cfg.services = append(cfg.services, desc)
	}
}

func WithExecutor(exec TaskExecutor) Option {
	return func(cfg *agentConfig) {
		cfg.Executor = exec
	}
}

func WithSessions(mgr *SessionManager) Option {
	return func(cfg *agentConfig) {
		cfg.Sessions = mgr
	}
}

func WithTools(reg ToolRegistry) Option {
	return func(cfg *agentConfig) {
		cfg.Tools = reg
	}
}

func WithSkills(reg SkillRegistry) Option {
	return func(cfg *agentConfig) {
		cfg.Skills = reg
	}
}

func WithMaxToolIterations(limit int) Option {
	return func(cfg *agentConfig) {
		cfg.maxToolIterations = limit
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(cfg *agentConfig) {
		cfg.logger = logger
	}
}

func WithStaticKnowledge(store StaticStore) Option {
	return func(cfg *agentConfig) {
		cfg.StaticKnowledge = store
	}
}

func WithRetriever(retriever Retriever) Option {
	return func(cfg *agentConfig) {
		cfg.Retriever = retriever
	}
}

// WithServerOption forwards options to the underlying rony.Server.
// Ignored when WithServer supplies an existing server.
func WithServerOption(opts ...rony.ServerOption) Option {
	return func(cfg *agentConfig) {
		cfg.serverOpts = append(cfg.serverOpts, opts...)
	}
}

// WithServer uses an existing rony.Server instead of creating one in New.
func WithServer(srv *rony.Server) Option {
	return func(cfg *agentConfig) {
		cfg.externalSrv = srv
	}
}
