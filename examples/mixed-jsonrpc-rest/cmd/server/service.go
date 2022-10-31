package main

import (
	"net/http"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/desc"
	"github.com/clubpay/ronykit/examples/simple-rest-server/dto"
	"github.com/clubpay/ronykit/std/gateway/fasthttp"
	"github.com/clubpay/ronykit/std/gateway/fastws"
)

var sample desc.ServiceDescFunc = func() *desc.Service {
	return desc.NewService("SampleService").
		SetEncoding(ronykit.JSON).
		AddContract(
			desc.NewContract().
				SetInput(&dto.EchoRequest{}).
				SetOutput(&dto.EchoResponse{}).
				AddSelector(fasthttp.REST(http.MethodGet, "/echo/:randomID")).
				AddSelector(fasthttp.RPC("echoRequest")).
				AddSelector(fastws.RPC("echoRequest")).
				SetHandler(echoHandler),
		)
}

func echoHandler(ctx *ronykit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(*dto.EchoRequest)

	ctx.In().Reply().
		SetHdr("cmd", ctx.In().GetHdr("cmd")).
		SetMsg(
			&dto.EchoResponse{
				RandomID: req.RandomID,
				Ok:       req.Ok,
			},
		).Send()

	return
}
