package rony

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

type (
	Action                         comparable
	State[A Action]                interface{}
	StatePtr[O State[A], A Action] interface {
		*O
		Name() string
		Reduce(action A)
	}
	Message kit.Message
	Error   kit.ErrorMessage
)

type Handler[
	A Action, S State[A], SP StatePtr[S, A],
	IN, OUT Message,
] func(ctx *Context[A, SP, S], in IN) (OUT, Error)

type RESTSelector kit.RESTRouteSelector

var (
	GET    = fasthttp.GET
	POST   = fasthttp.POST
	PUT    = fasthttp.PUT
	DELETE = fasthttp.DELETE
	PATCH  = fasthttp.PATCH
)
