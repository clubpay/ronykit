package main

import (
	"net/http"

	"github.com/clubpay/ronykit/example/mixed-jsonrpc-rest/cmd/dto"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/std/gateways/fastws"
)

var sample desc.ServiceDescFunc = func() *desc.Service {
	return desc.NewService("SampleService").
		SetEncoding(kit.JSON).
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

func echoHandler(ctx *kit.Context) {
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
