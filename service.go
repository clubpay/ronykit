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
	Name() string
	Routes() []Route
	PreHandlers() []Handler
	PostHandlers() []Handler
}

// Route defines the set of Handlers based on the Query. Query is different per bundles,
// hence, this is the implementor's task to make sure return correct value based on 'q'
type Route interface {
	Query(q string) interface{}
	Handlers() []Handler
}

// stdService is a simple implementation of Service interface.
type stdService struct {
	name   string
	pre    []Handler
	post   []Handler
	routes []Route
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

func (s *stdService) Routes() []Route {
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

func (s *stdService) AddRoute(routeParams map[string]interface{}, handlers ...Handler) *stdService {
	s.routes = append(
		s.routes,
		&route{
			routeParams: routeParams,
			handlers:    handlers,
		},
	)

	return s
}

type route struct {
	routeParams map[string]interface{}
	handlers    []Handler
}

func (r *route) Query(q string) interface{} {
	return r.routeParams[q]
}

func (r *route) Handlers() []Handler {
	return r.handlers
}
