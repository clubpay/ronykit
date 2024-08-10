package kit

// RouteSelector holds information about how this Contract is going to be selected. Each
// Gateway may need different information to route the request to the right Contract.
// RouteSelector is actually a base interface, and Gateway implementors usually implement
// either RESTRouteSelector, RPCRouteSelector, or both. It is an interface that provides methods
// for querying and getting the encoding of a routing object. Implementations of RouteSelector are
// used by Gateway to handle different types of request routing based on the method and protocol. The
// Query method is for searching specific routes with a query string, and GetEncoding is for
// returning the encoding format required for communication between the gateway and the target contract.
type RouteSelector interface {
	Query(q string) any
	GetEncoding() Encoding
}

// RESTRouteSelector is an interface that extends RouteSelector with methods specific to REST operations,
// primarily used by REST-based gateways. With GetMethod and GetPath methods, this interface provides
// the HTTP method and path information necessary for routing REST requests to the appropriate contract.
// Implementations of this interface are expected to provide their own GetMethod and GetPath methods that
// return the desired method and path as strings.
type RESTRouteSelector interface {
	RouteSelector
	GetMethod() string
	GetPath() string
}

// RPCRouteSelector defines the RouteSelector, which could be used in RPC operations.
// Gateway implementation that handles RPC requests could check the selector if it supports RPC.
// It is an interface that extends RouteSelector with a method specific to RPC operations. It is primarily
// used by RPC-based gateways for routing RPC requests to the appropriate contract. Implementations of this
// interface should provide their own GetPredicate method, which returns a string representing a predicate
// used to evaluate the route. In this context, a predicate is a condition or criteria that helps in
// selecting the proper handler for the RPC request.
type RPCRouteSelector interface {
	RouteSelector
	GetPredicate() string
}

// EdgeSelectorFunc returns the target EdgeServer's ID in the Cluster. If you have
// multiple instances of the EdgeServer, and you want to forward some requests to a specific
// instance, you can set up this function in desc.Contract's SetCoordinator method, then, the
// receiving EdgeServer will detect the target instance by calling this function and forwards
// the request to the returned instance.
// From the external client point of view this forwarding request is not observable.
type EdgeSelectorFunc func(ctx *LimitedContext) (string, error)

// Contract defines the set of Handlers based on the Query. Query is different per bundles,
// hence, this is the implementor's task to make sure return the correct value based on 'q'.
// In other words, Contract 'r' must return a valid response for 'q's required by Gateway 'gw' in
// order to be usable by Gateway 'gw' otherwise it panics.
type Contract interface {
	// ID identifies the contract. This MUST be unique per Service. This MUST NOT be a runtime
	// random number. Since this is used in the RemoteExecute method of ClusterMember to execute the
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
	Output() Message
	Handlers() []HandlerFunc
	Modifiers() []ModifierFunc
}

// ContractWrapper is like an interceptor which can add Pre- and Post- handlers to all
// the Contracts of the Contract.
type ContractWrapper interface {
	Wrap(c Contract) Contract
}

// ContractWrapperFunc implements ContractWrapper interface.
type ContractWrapperFunc func(Contract) Contract

func (sw ContractWrapperFunc) Wrap(svc Contract) Contract {
	return sw(svc)
}

// WrapContract wraps a contract, this is useful for adding middlewares to the contract.
// Some middlewares like OpenTelemetry, Logger, and so on could be added to the contract using
// this function.
func WrapContract(c Contract, wrappers ...ContractWrapper) Contract {
	for _, w := range wrappers {
		c = w.Wrap(c)
	}

	return c
}

// contractWrap implements the Contract interface and is useful when we need to wrap another
// contract.
type contractWrap struct {
	Contract
	h     []HandlerFunc
	preM  []ModifierFunc
	postM []ModifierFunc
}

var _ Contract = (*contractWrap)(nil)

func (c contractWrap) Handlers() []HandlerFunc {
	h := make([]HandlerFunc, 0, len(c.h)+len(c.Contract.Handlers()))
	h = append(h, c.h...)
	h = append(h, c.Contract.Handlers()...)

	return h
}

func (c contractWrap) Modifiers() []ModifierFunc {
	m := make([]ModifierFunc, 0, len(c.preM)+len(c.postM)+len(c.Contract.Modifiers()))
	m = append(m, c.preM...)
	m = append(m, c.Contract.Modifiers()...)
	m = append(m, c.postM...)

	return m
}
