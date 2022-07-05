package main

import (
	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/desc"
	"github.com/clubpay/ronykit/exmples/simple-rest-server/dto"
	"github.com/clubpay/ronykit/std/gateway/fasthttp"
	"github.com/clubpay/ronykit/std/gateway/fastws"
)

type Sample struct{}

func NewSample() *Sample {
	return &Sample{}
}

func (s *Sample) Desc() *desc.Service {
	return desc.NewService("SampleService").
		AddContract(
			desc.NewContract().
				SetInput(&dto.EchoRequest{}).
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
	req, ok := ctx.In().GetMsg().(*dto.EchoRequest)
	if !ok {
		ctx.Out().
			SetMsg(ronykit.RawMessage("Request was not echoRequest")).
			Send()

		return
	}

	ctx.Out().
		SetHdr("cmd", ctx.In().GetHdr("cmd")).
		SetMsg(
			&dto.EchoResponse{
				RandomID: req.RandomID,
			},
		).Send()

	return
}
