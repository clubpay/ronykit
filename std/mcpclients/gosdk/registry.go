package gosdk

import (
	"context"
	"fmt"
	"sync"

	"github.com/clubpay/ronykit/intent"
)

// Registry holds MCP servers created from configuration.
type Registry struct {
	mu      sync.RWMutex
	factory intent.MCPClientFactory
	configs []intent.MCPServerConfig
	servers map[string]intent.MCPServer
}

// NewRegistry returns a registry backed by factory.
func NewRegistry(factory intent.MCPClientFactory, configs ...intent.MCPServerConfig) *Registry {
	return &Registry{
		factory: factory,
		configs: append([]intent.MCPServerConfig(nil), configs...),
		servers: make(map[string]intent.MCPServer),
	}
}

func (r *Registry) List() []intent.MCPServerConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return append([]intent.MCPServerConfig(nil), r.configs...)
}

func (r *Registry) Get(name string) (intent.MCPServer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	srv, ok := r.servers[name]

	return srv, ok
}

func (r *Registry) ConnectAll(ctx context.Context) error {
	if r.factory == nil {
		return fmt.Errorf("mcp registry factory is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, cfg := range r.configs {
		if _, ok := r.servers[cfg.Name]; ok {
			continue
		}

		srv, err := r.factory.NewServer(ctx, cfg)
		if err != nil {
			return err
		}

		r.servers[cfg.Name] = srv
	}

	return nil
}

func (r *Registry) CloseAll() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var firstErr error

	for name, srv := range r.servers {
		err := srv.Close()
		if err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close mcp server %q: %w", name, err)
		}

		delete(r.servers, name)
	}

	return firstErr
}

var _ intent.MCPRegistry = (*Registry)(nil)
