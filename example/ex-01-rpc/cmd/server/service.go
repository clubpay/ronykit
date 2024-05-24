package main

import (
	"net/http"

	"github.com/clubpay/ronykit/example/ex-01-rpc/dto"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/std/gateways/fastws"
)

var sampleService = desc.NewService("SampleService").
	SetEncoding(kit.JSON).
	AddContract(
		desc.NewContract().
			SetInput(&dto.EchoRequest{}).
			SetOutput(&dto.EchoResponse{}).
			AddSelector(fasthttp.REST(http.MethodGet, "/echo/:randomID")).
			AddSelector(fasthttp.RPCs("echoRequest")...).
			AddSelector(fastws.RPCs("echoRequest")...).
			SetHandler(echoHandler),
	)

func echoHandler(ctx *kit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(*dto.EchoRequest)

	// fmt.Println("got request: ", req)
	ctx.In().Reply().
		SetHdr("cmd", ctx.In().GetHdr("cmd")).
		SetMsg(
			&dto.EchoResponse{
				RandomID: req.RandomID,
				Ok:       req.Ok,
				Ts:       utils.NanoTime(),
			},
		).Send()
}
