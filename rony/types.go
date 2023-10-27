package rony

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

type (
	Action          comparable
	State[A Action] interface {
		Name() string
		Reduce(action A)
	}
	Message kit.Message
	Error   kit.ErrorMessage
)

type Handler[
	A Action, S State[A],
	IN, OUT Message,
] func(ctx *Context[A, S], in IN) (OUT, Error)

type RESTSelector kit.RESTRouteSelector

var (
	GET    = fasthttp.GET
	POST   = fasthttp.POST
	PUT    = fasthttp.PUT
	DELETE = fasthttp.DELETE
	PATCH  = fasthttp.PATCH
)
