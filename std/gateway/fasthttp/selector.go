package fasthttp

import (
	"github.com/clubpay/ronykit"
)

// Selector implements ronykit.RouteSelector and
// also ronykit.RPCRouteSelector and ronykit.RESTRouteSelector
type Selector struct {
	Method    string
	Path      string
	Predicate string
	Decoder   DecoderFunc
	Encoding  ronykit.Encoding
}

var (
	_ ronykit.RouteSelector     = (*Selector)(nil)
	_ ronykit.RESTRouteSelector = (*Selector)(nil)
	_ ronykit.RPCRouteSelector  = (*Selector)(nil)
)

func (r Selector) GetEncoding() ronykit.Encoding {
	return r.Encoding
}

func (r Selector) GetMethod() string {
	return r.Method
}

func (r Selector) GetPath() string {
	return r.Path
}

func (r Selector) GetPredicate() string {
	return r.Predicate
}

func (r Selector) Query(q string) interface{} {
	switch q {
	case queryDecoder:
		return r.Decoder
	case queryMethod:
		return r.Method
	case queryPath:
		return r.Path
	case queryPredicate:
		return r.Predicate
	}

	return nil
}
