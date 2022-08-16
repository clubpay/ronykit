package fasthttp

import (
	"net/http"

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

// REST returns a Selector which acts on http requests.
func REST(method, path string) Selector {
	return Selector{
		Method: method,
		Path:   path,
	}
}

// POST a shortcut for REST(http.MethodPost, path)
func POST(path string) Selector {
	return REST(http.MethodPost, path)
}

// GET a shortcut for REST(http.MethodGet, path)
func GET(path string) Selector {
	return REST(http.MethodGet, path)
}

// PATCH a shortcut for REST(http.MethodPatch, path)
func PATCH(path string) Selector {
	return REST(http.MethodPatch, path)
}

// PUT a shortcut for REST(http.MethodPut, path)
func PUT(path string) Selector {
	return REST(http.MethodPut, path)
}

// DELETE a shortcut for REST(http.MethodDelete, path)
func DELETE(path string) Selector {
	return REST(http.MethodDelete, path)
}

// RPC returns a Selector which acts on websocket requests
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
