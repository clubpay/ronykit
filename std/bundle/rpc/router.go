package rpc

import (
	"github.com/ronaksoft/ronykit"
)

type DecoderFunc func(data []byte, e *Envelope) error

type routerData struct {
	ServiceName string
	Handlers    []ronykit.Handler
	Factory     func() ronykit.Message
}

type router struct {
	routes map[string]routerData
}

type routeData struct {
	predicate string
	factory   ronykit.MessageFactory
}

func Route(predicate string, factory func() ronykit.Message) *routeData {
	return &routeData{
		predicate: predicate,
		factory:   factory,
	}
}

func (r routeData) Query(q string) interface{} {
	switch q {
	case queryPredicate:
		return r.predicate
	case queryFactory:
		return r.factory
	}

	return nil
}
