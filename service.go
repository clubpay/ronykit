package ronykit

// ServiceWrapper is like an interceptor which can add Pre- and Post- handlers to the
// Contract handlers.
type ServiceWrapper func(Service) Service

// WrapService wraps a service, this is useful for adding middlewares to the service.
// Some middlewares like OpenTelemetry, Logger, ... could be added to the service using
// this function.
func WrapService(svc Service, wrappers ...ServiceWrapper) Service {
	for _, w := range wrappers {
		svc = w(svc)
	}

	return svc
}

// Service defines a set of RPC handlers which usually they are related to one service.
// Name must be unique per each Bundle.
type Service interface {
	// Name of the service which must be unique per Server.
	Name() string
	// Contracts returns a list of APIs which this service provides.
	Contracts() []Contract
	// PreHandlers returns a list of handlers which are executed in sequence **BEFORE** any
	// call. If you need specific pre-handlers per Contract, then should be defined the
	// Contract itself.
	PreHandlers() []Handler
	// PostHandlers returns a list of handlers which are executed in sequence **AFTER** any
	// call. If you need specific pre-handlers per Contract, then should be defined the
	// Contract itself.
	PostHandlers() []Handler
}

// Contract defines the set of Handlers based on the Query. Query is different per bundles,
// hence, this is the implementor's task to make sure return correct value based on 'q'.
// In other words, Contract 'r' must return valid response for 'q's required by Bundle 'b' in
// order to be usable by Bundle 'b' otherwise it panics.
type Contract interface {
	RouteSelector
	Input() Message
	Handlers() []Handler
	Modifiers() []Modifier
}

// RouteSelector holds information about how this Contract is going to be selected. Each
// Bundle may need different information to route the request to the right Contract.
type RouteSelector interface {
	Query(q string) interface{}
}

// RESTRouteSelector defines the RouteSelector which could be used in REST operations
// Bundle implementation which handle REST requests could check the selector if it supports REST.
type RESTRouteSelector interface {
	RouteSelector
	GetMethod() string
	GetPath() string
}

// RPCRouteSelector defines the RouteSelector which could be used in RPC operations.
// Bundle implementation which handle RPC requests could check the selector if it supports RPC
type RPCRouteSelector interface {
	RouteSelector
	GetPredicate() string
}
