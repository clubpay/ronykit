package ronykit

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
	// PreHandlers returns a list of handlers which are executed in sequence BEFORE any
	// call. If you need specific pre-handlers per Contract, then should be defined the
	// Contract itself.
	PreHandlers() []Handler
	// PostHandlers returns a list of handlers which are executed in sequence AFTER any
	// call. If you need specific pre-handlers per Contract, then should be defined the
	// Contract itself.
	PostHandlers() []Handler
}

// Contract defines the set of Handlers based on the Query. Query is different per bundles,
// hence, this is the implementor's task to make sure return correct value based on 'q'.
// In other words, Contract 'r' must return valid response for 'q's required by Bundle 'b' in
// order to be usable by Bundle 'b' otherwise it panics.
type Contract interface {
	RouteInfo
	Handlers() []Handler
}

// RouteInfo holds information about how this Contract is going to be selected. Each
// Bundle may need different information to route the request to the right Contract.
type RouteInfo interface {
	Query(q string) interface{}
}

// stdService is a simple implementation of Service interface.
type stdService struct {
	name   string
	pre    []Handler
	post   []Handler
	routes []Contract
}

func NewService(name string) *stdService {
	s := &stdService{
		name: name,
	}

	return s
}

func (s *stdService) Name() string {
	return s.name
}

func (s *stdService) Contracts() []Contract {
	return s.routes
}

func (s *stdService) PreHandlers() []Handler {
	return s.pre
}

func (s *stdService) PostHandlers() []Handler {
	return s.post
}

func (s *stdService) SetPreHandlers(h ...Handler) *stdService {
	s.pre = append(s.pre[:0], h...)

	return s
}

func (s *stdService) SetPostHandlers(h ...Handler) *stdService {
	s.post = append(s.post[:0], h...)

	return s
}

func (s *stdService) AddContract(r Contract) *stdService {
	s.routes = append(s.routes, r)

	return s
}

// simpleContract is simple implementation of Contract interface.
type simpleContract struct {
	routeData []RouteInfo
	handlers  []Handler
}

func NewContract() *simpleContract {
	return &simpleContract{}
}

func (r *simpleContract) AddRoute(rd ...RouteInfo) *simpleContract {
	r.routeData = append(r.routeData, rd...)

	return r
}

func (r *simpleContract) Query(q string) interface{} {
	for _, rd := range r.routeData {
		v := rd.Query(q)
		if v != nil {
			return v
		}
	}

	return nil
}

func (r *simpleContract) SetHandler(handlers ...Handler) *simpleContract {
	r.handlers = append(r.handlers[:0], handlers...)

	return r
}

func (r *simpleContract) Handlers() []Handler {
	return r.handlers
}
