package fastws

import (
	"github.com/clubpay/ronykit"
)

type Selector struct {
	Predicate string
	Encoding  ronykit.Encoding
}

var _ ronykit.RPCRouteSelector = (*Selector)(nil)

// RPC returns a selector that acts on websocket messages
func RPC(predicate string) Selector {
	return Selector{
		Predicate: predicate,
	}
}

func (r Selector) GetEncoding() ronykit.Encoding {
	return r.Encoding
}

func (r *Selector) SetEncoding(enc ronykit.Encoding) *Selector {
	r.Encoding = enc

	return r
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

type routeData struct {
	ServiceName string
	Predicate   string
	ContractID  string
	Factory     ronykit.MessageFactoryFunc
}
