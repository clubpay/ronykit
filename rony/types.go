package rony

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

// Alias types
type (
	Message          kit.Message
	Selector         kit.RouteSelector
	RESTParams       = fasthttp.Params
	RESTParam        = fasthttp.Param
	NodeSelectorFunc = kit.EdgeSelectorFunc
)
