package fasthttp

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/clubpay/ronykit/kit"
)

// Selector implements kit.RouteSelector and
// also kit.RPCRouteSelector and kit.RESTRouteSelector
type Selector struct {
	Method    string
	Path      string
	Predicate string
	Decoder   DecoderFunc
	Encoding  kit.Encoding
}

var (
	_ kit.RouteSelector     = (*Selector)(nil)
	_ kit.RESTRouteSelector = (*Selector)(nil)
	_ kit.RPCRouteSelector  = (*Selector)(nil)
)

var legacyPathFormatRegEx = regexp.MustCompile(`/:([-_a-zA-Z0-9]*)[^/]`)

func convertLegacyPathFormat(path string) string {
	return legacyPathFormatRegEx.ReplaceAllStringFunc(
		path,
		func(s string) string {
			return fmt.Sprintf("/{%s}", strings.TrimPrefix(s, "/:"))
		},
	)
}

// REST returns a Selector which acts on http requests.
func REST(method, path string) Selector {
	path = convertLegacyPathFormat(path)

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

// RPCs is a shortcut for multiple RPC selectors
func RPCs(predicate ...string) []kit.RouteSelector {
	selectors := make([]kit.RouteSelector, 0, len(predicate))
	for idx := range predicate {
		selectors = append(selectors, RPC(predicate[idx]))
	}

	return selectors
}

func (r Selector) GetEncoding() kit.Encoding {
	return r.Encoding
}

func (r Selector) SetEncoding(enc kit.Encoding) Selector {
	r.Encoding = enc

	return r
}

func (r Selector) SetDecoder(f DecoderFunc) Selector {
	r.Decoder = f

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

func (r Selector) String() string {
	if r.Predicate != "" {
		return r.Predicate
	}

	return fmt.Sprintf("%s %s", r.Method, r.Path)
}
