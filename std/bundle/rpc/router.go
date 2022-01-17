package rpc

import (
	"github.com/ronaksoft/ronykit"
)

type DecoderFunc func(data []byte, e *Envelope) error

type routerData struct {
	ServiceName string
	Handlers    []ronykit.Handler
	Factory     ronykit.MessageFactory
}

type mux struct {
	routes map[string]routerData
}

type routeSelector struct {
	predicate string
	factory   ronykit.MessageFactory
}

func Route(predicate string) *routeSelector {
	return &routeSelector{
		predicate: predicate,
	}
}

func (r *routeSelector) WithFactory(f ronykit.MessageFactory) *routeSelector {
	r.factory = f

	return r
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
