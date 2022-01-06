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
// hence, this is the implementor's task to make sure return correct value based on 'q'.
// In other words, Route 'r' must return valid response for 'q's required by Bundle 'b' in
// order to be usable by Bundle 'b' otherwise it panics.
type Route interface {
	RouteData
	Handlers() []Handler
}

type RouteData interface {
	Query(q string) interface{}
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

func (s *stdService) AddRoute(r Route) *stdService {
	s.routes = append(s.routes, r)

	return s
}

// stdRoute is simple implementation of Route interface.
type stdRoute struct {
	routeData []RouteData
	handlers  []Handler
}

func NewRoute() *stdRoute {
	return &stdRoute{}
}

func (r *stdRoute) SetData(rd RouteData) *stdRoute {
	r.routeData = append(r.routeData, rd)

	return r
}

func (r *stdRoute) Query(q string) interface{} {
	for _, rd := range r.routeData {
		v := rd.Query(q)
		if v != nil {
			return v
		}
	}

	return nil
}

func (r *stdRoute) SetHandler(handlers ...Handler) *stdRoute {
	r.handlers = append(r.handlers[:0], handlers...)

	return r
}

func (r *stdRoute) Handlers() []Handler {
	return r.handlers
}
