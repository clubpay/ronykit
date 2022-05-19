package fastws

import (
	"github.com/clubpay/ronykit"
)

type Selector struct {
	Predicate string
}

var _ ronykit.RPCRouteSelector = (*Selector)(nil)

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

type routeData struct {
	ServiceName string
	Predicate   string
	ContractID  string
	Factory     ronykit.MessageFactoryFunc
}
