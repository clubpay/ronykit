package main

import (
	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/desc"
	"github.com/clubpay/ronykit/exmples/mixed-jsonrpc-rest/msg"
	"github.com/clubpay/ronykit/std/bundle/fasthttp"
	"github.com/clubpay/ronykit/std/bundle/fastws"
)

type Sample struct{}

func NewSample() *Sample {
	return &Sample{}
}

func (s *Sample) Desc() *desc.Service {
	return desc.NewService("SampleService").
		AddContract(
			desc.NewContract().
				SetInput(&msg.EchoRequest{}).
				AddSelector(fasthttp.Selector{
					Method:    fasthttp.MethodGet,
					Path:      "/echo/:randomID",
					Predicate: "echoRequest",
				}).
				AddSelector(fastws.Selector{
					Predicate: "echoRequest",
				}).
				SetHandler(echoHandler),
		)
}

func echoHandler(ctx *ronykit.Context) {
	req, ok := ctx.In().GetMsg().(*msg.EchoRequest)
	if !ok {
		ctx.Out().
			SetMsg(ronykit.RawMessage("Request was not echoRequest")).
			Send()

		return
	}

	ctx.Out().
		SetMsg(
			&msg.EchoResponse{
				RandomID: req.RandomID,
			},
		).Send()

	return
}
