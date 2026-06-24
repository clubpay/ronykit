package intent

import (
	"fmt"
	"sync"
)

// MapServiceRegistry is an in-memory service registry.
type MapServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]ServiceDescriptor
}

// NewMapServiceRegistry returns an empty service registry.
func NewMapServiceRegistry() *MapServiceRegistry {
	return &MapServiceRegistry{services: make(map[string]ServiceDescriptor)}
}

func (r *MapServiceRegistry) Register(desc ServiceDescriptor) error {
	if desc.Name == "" {
		return fmt.Errorf("service name is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.services[desc.Name] = desc

	return nil
}

func (r *MapServiceRegistry) Get(name string) (ServiceDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	desc, ok := r.services[name]

	return desc, ok
}

func (r *MapServiceRegistry) All() []ServiceDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]ServiceDescriptor, 0, len(r.services))
	for _, desc := range r.services {
		out = append(out, desc)
	}

	return out
}

var _ ServiceRegistry = (*MapServiceRegistry)(nil)
