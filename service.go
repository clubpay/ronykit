package ronykit

type ServiceWrapper func(IService) IService

// WrapService wraps a service, this is useful for adding middlewares to the service.
// Some middlewares like OpenTelemetry, Logger, ... could be added to the service using
// this function.
func WrapService(svc IService, wrappers ...ServiceWrapper) IService {
	for _, w := range wrappers {
		svc = w(svc)
	}

	return svc
}

// IService defines a set of RPC handlers which usually they are related to one service.
// Name must be unique per each Bundle.
type IService interface {
	Name() string
	Routes() []IRoute
	PreHandlers() []Handler
	PostHandlers() []Handler
}

// IRoute defines the set of Handlers based on the Query. Query is different per bundles,
// hence, this is the implementor's task to make sure return correct value based on 'q'
type IRoute interface {
	Query(q string) interface{}
	Handlers() []Handler
}

type Service struct {
	name   string
	pre    []Handler
	post   []Handler
	routes []IRoute
}

func NewService(name string) *Service {
	s := &Service{
		name: name,
	}

	return s
}

func (s *Service) Name() string {
	return s.name
}

func (s *Service) Routes() []IRoute {
	return s.routes
}

func (s *Service) PreHandlers() []Handler {
	return s.pre
}

func (s *Service) PostHandlers() []Handler {
	return s.post
}

func (s *Service) SetPreHandlers(h ...Handler) *Service {
	s.pre = append(s.pre[:0], h...)

	return s
}

func (s *Service) SetPostHandlers(h ...Handler) *Service {
	s.post = append(s.post[:0], h...)

	return s
}

func (s *Service) AddRoute(routeParams map[string]interface{}, handlers ...Handler) *Service {
	s.routes = append(
		s.routes,
		&Route{
			routeParams: routeParams,
			handlers:    handlers,
		},
	)

	return s
}

type Route struct {
	routeParams map[string]interface{}
	handlers    []Handler
}

func (r *Route) Query(q string) interface{} {
	return r.routeParams[q]
}

func (r *Route) Handlers() []Handler {
	return r.handlers
}
