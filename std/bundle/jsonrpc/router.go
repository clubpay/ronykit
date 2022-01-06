package jsonrpc

import (
	"github.com/ronaksoft/ronykit"
)

type routerData struct {
	ServiceName string
	Handlers    []ronykit.Handler
}

type router struct {
	routes map[string]routerData
}

type routeData struct {
	predicate string
}

func NewRouteData(predicate string) *routeData {
	return &routeData{
		predicate: predicate,
	}
}

func (r routeData) Query(q string) interface{} {
	switch q {
	case queryPredicate:
		return r.predicate
	}

	return nil
}
