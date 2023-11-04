package rony

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

// State related types
type (
	Action          comparable
	State[A Action] interface {
		Name() string
		Reduce(action A)
	}
)

// Alias types
type (
	Message    kit.Message
	Error      kit.ErrorMessage
	Selector   kit.RouteSelector
	RESTParams = fasthttp.Params
	RESTParam  = fasthttp.Param
)
