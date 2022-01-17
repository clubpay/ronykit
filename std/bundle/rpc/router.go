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

func Selector(predicate string, factory ronykit.MessageFactory) *routeSelector {
	return &routeSelector{
		predicate: predicate,
		factory:   factory,
	}
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
