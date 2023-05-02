package kit

type ServiceDescriptor interface {
	Generate() Service
}

// Service defines a set of RPC handlers which usually they are related to one service.
// Name must be unique per each Gateway.
type Service interface {
	// Name of the service which must be unique per EdgeServer.
	Name() string
	// Contracts return a list of APIs which this service provides.
	Contracts() []Contract
}

// ServiceWrapper lets you add customizations to your service. A specific case of it is serviceInterceptor
// which can add Pre- and Post-handlers to all the Contracts of the Service.
type ServiceWrapper interface {
	Wrap(Service) Service
}

type ServiceWrapperFunc func(Service) Service

func (sw ServiceWrapperFunc) Wrap(svc Service) Service {
	return sw(svc)
}

// WrapService wraps a service, this is useful for adding middlewares to the service.
// Some middlewares like OpenTelemetry, Logger, ... could be added to the service using
// this function.
func WrapService(svc Service, wrappers ...ServiceWrapper) Service {
	for _, w := range wrappers {
		svc = w.Wrap(svc)
	}

	return svc
}

func WrapServiceContracts(svc Service, wrapper ...ContractWrapper) Service {
	sw := &serviceWrap{
		name: svc.Name(),
	}

	for _, c := range svc.Contracts() {
		for _, w := range wrapper {
			c = w.Wrap(c)
		}
		sw.contracts = append(sw.contracts, c)
	}

	return sw
}

// serviceWrap implements Service interface and is useful when we need to wrap another
// service.
type serviceWrap struct {
	name      string
	contracts []Contract
}

var _ Service = (*serviceWrap)(nil)

func (sw serviceWrap) Contracts() []Contract {
	return sw.contracts
}

func (sw serviceWrap) Name() string {
	return sw.name
}
