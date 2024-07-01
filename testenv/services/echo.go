package services

import (
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fastws"
)

type EchoRequest struct {
	Input string `json:"input"`
}

type EchoResponse struct {
	Output string `json:"output"`
}

var EchoService kit.ServiceBuilder = desc.NewService("EchoService").
	AddContract(
		desc.NewContract().
			SetInput(&EchoRequest{}).
			SetOutput(&EchoResponse{}).
			Selector(fastws.RPC("echo")).
			SetHandler(
				contextMW(10*time.Second),
				func(ctx *kit.Context) {
					req, _ := ctx.In().GetMsg().(*EchoRequest)

					ctx.Out().
						SetMsg(&EchoResponse{Output: req.Input}).
						Send()
				}),
	)
