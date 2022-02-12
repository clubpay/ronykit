package fastws

import (
	"github.com/ronaksoft/ronykit"
)

type Selector struct {
	Predicate string
}

func (s Selector) Generate(f ronykit.MessageFactory) ronykit.RouteSelector {
	return &routeSelector{
		predicate: s.Predicate,
		factory:   f,
	}
}

type routeSelector struct {
	predicate string
	factory   ronykit.MessageFactory
}

func (r routeSelector) Query(q string) interface{} {
	switch q {
	case queryPredicate:
		return r.predicate
	case queryFactory:
		return r.factory
	}

	return nil
}

type routerData struct {
	ServiceName string
	Predicate   string
	Handlers    []ronykit.Handler
	Modifiers   []ronykit.Modifier
	Factory     ronykit.MessageFactory
}

type mux struct {
	routes map[string]routerData
}
