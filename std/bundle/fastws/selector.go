package fastws

import (
	"github.com/ronaksoft/ronykit"
)

var (
	_ ronykit.RouteSelector    = Selector{}
	_ ronykit.RPCRouteSelector = Selector{}
)

type Selector struct {
	Predicate string
}

func (r Selector) GetPredicate() string {
	return r.Predicate
}

func (r Selector) Query(q string) interface{} {
	switch q {
	case queryPredicate:
		return r.Predicate
	}

	return nil
}

type routerData struct {
	ServiceName string
	Predicate   string
	Handlers    []ronykit.Handler
	Modifiers   []ronykit.Modifier
	Factory     ronykit.MessageFactoryFunc
}

type mux struct {
	routes map[string]routerData
}
