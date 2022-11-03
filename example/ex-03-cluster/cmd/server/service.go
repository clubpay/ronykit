package main

import (
	"net/http"

	"github.com/clubpay/ronykit/example/ex-03-cluster/dto"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

func genDesc(instanceID string) desc.ServiceDescFunc {
	return func() *desc.Service {
		return desc.NewService("SampleService").
			SetEncoding(kit.JSON).
			AddContract(
				desc.NewContract().
					SetCoordinator(coordinator).
					SetInput(&dto.EchoRequest{}).
					SetOutput(&dto.EchoResponse{}).
					AddSelector(fasthttp.REST(http.MethodGet, "/echo/:randomID")).
					SetHandler(genEchoHandler(instanceID)),
			)
	}
}

func coordinator(ctx *kit.LimitedContext) (string, error) {
	return ctx.In().GetHdr("Redirect-To"), nil
}

func genEchoHandler(instanceID string) kit.HandlerFunc {
	return func(ctx *kit.Context) {
		//nolint:forcetypeassert
		req := ctx.In().GetMsg().(*dto.EchoRequest)

		ctx.In().Reply().
			SetHdr("cmd", ctx.In().GetHdr("cmd")).
			SetMsg(
				&dto.EchoResponse{
					RandomID: req.RandomID,
					OriginID: instanceID,
				},
			).Send()

		return
	}
}
