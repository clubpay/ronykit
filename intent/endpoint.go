package intent

import (
	"github.com/clubpay/ronykit/rony"
)

// EndpointMount is the registration surface agents use to define rony services
// without holding a *rony.Server reference. The agent passes a mount during
// construction; implementations in std wire handlers through rony.Setup.
type EndpointMount struct {
	srv *rony.Server
}

// Setup registers a rony service on the agent server.
func Setup[S rony.State[A], A rony.Action](
	m EndpointMount,
	serviceName string,
	init rony.InitState[S, A],
	opts ...rony.SetupOption[S, A],
) {
	if m.srv == nil || init == nil {
		return
	}

	rony.Setup(m.srv, serviceName, init, opts...)
}

// ServiceDescriptor binds a named rony service to the agent server.
type ServiceDescriptor struct {
	Name  string
	Mount func(m EndpointMount) error
}

// ServiceRegistry holds service descriptors registered on an agent.
type ServiceRegistry interface {
	Register(desc ServiceDescriptor) error
	Get(name string) (ServiceDescriptor, bool)
	All() []ServiceDescriptor
}
