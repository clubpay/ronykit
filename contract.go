package ronykit

// RouteSelector holds information about how this Contract is going to be selected. Each
// Gateway may need different information to route the request to the right Contract.
// RouteSelector is actually a base interface and Gateway implementors usually implement
// either RESTRouteSelector, RPCRouteSelector or both.
type RouteSelector interface {
	Query(q string) interface{}
	GetEncoding() Encoding
}

// RESTRouteSelector defines the RouteSelector which could be used in REST operations
// Gateway implementation which handle REST requests could check the selector if it supports REST.
type RESTRouteSelector interface {
	RouteSelector
	GetMethod() string
	GetPath() string
}

// RPCRouteSelector defines the RouteSelector which could be used in RPC operations.
// Gateway implementation which handle RPC requests could check the selector if it supports RPC
type RPCRouteSelector interface {
	RouteSelector
	GetPredicate() string
}

// EdgeSelectorFunc returns the target EdgeServer in the Cluster. If you have
// multiple instances of the EdgeServer, and you want to forward some requests to a specific
// instance, you can set up this function in desc.Contract's SetCoordinator method, then, the
// receiving EdgeServer will detect the target instance by calling this function and forwards
// the request to the returned instance.
// From the external client point of view this forwarding request is not observable.
type EdgeSelectorFunc func(ctx *LimitedContext) (ClusterMember, error)

// Contract defines the set of Handlers based on the Query. Query is different per bundles,
// hence, this is the implementor's task to make sure return correct value based on 'q'.
// In other words, Contract 'r' must return valid response for 'q's required by Gateway 'gw' in
// order to be usable by Gateway 'gw' otherwise it panics.
type Contract interface {
	// ID identifies the contract. This MUST be unique per Service. This MUST NOT be a runtime
	// random number. Since this is used in RemoteExecute method of ClusterMember to execute the
	// right set of handlers on remote EdgeServer.
	ID() string
	// RouteSelector returns a RouteSelector function which selects this Contract based on the
	// client requests.
	RouteSelector() RouteSelector
	// EdgeSelector returns an EdgeSelectorFunc function which selects the EdgeServer instance
	// that the request should forward to for execution.
	EdgeSelector() EdgeSelectorFunc
	Encoding() Encoding
	Input() Message
	Handlers() []HandlerFunc
	Modifiers() []Modifier
}

// ContractWrapper is like an interceptor which can add Pre- and Post- handlers to all
// the Contracts of the Contract.
type ContractWrapper interface {
	Wrap(Contract) Contract
}

// ContractWrapperFunc implements ContractWrapper interface.
type ContractWrapperFunc func(Contract) Contract

func (sw ContractWrapperFunc) Wrap(svc Contract) Contract {
	return sw(svc)
}

// WrapContract wraps a contract, this is useful for adding middlewares to the contract.
// Some middlewares like OpenTelemetry, Logger, ... could be added to the contract using
// this function.
func WrapContract(c Contract, wrappers ...ContractWrapper) Contract {
	for _, w := range wrappers {
		c = w.Wrap(c)
	}

	return c
}

// contractWrap implements Contract interface and is useful when we need to wrap another
// contract.
type contractWrap struct {
	Contract
	h     []HandlerFunc
	preM  []Modifier
	postM []Modifier
}

var _ Contract = (*contractWrap)(nil)

func (c contractWrap) Handlers() []HandlerFunc {
	h := make([]HandlerFunc, 0, len(c.h)+len(c.Contract.Handlers()))
	h = append(h, c.h...)
	h = append(h, c.Contract.Handlers()...)

	return h
}

func (c contractWrap) Modifiers() []Modifier {
	m := make([]Modifier, 0, len(c.preM)+len(c.postM)+len(c.Contract.Modifiers()))
	m = append(m, c.preM...)
	m = append(m, c.Contract.Modifiers()...)
	m = append(m, c.postM...)

	return m
}
