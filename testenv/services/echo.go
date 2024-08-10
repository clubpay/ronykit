package services

import (
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/std/gateways/fastws"
)

type EchoRequest struct {
	Embedded
	Input string `json:"input"`
}

type EchoResponse struct {
	Embedded
	Output string `json:"output"`
}

type Embedded struct {
	X  string  `json:"x"`
	XP string  `json:"xp"`
	Y  int64   `json:"y"`
	Z  float64 `json:"z"`
	A  []byte  `json:"a"`
}

var EchoService kit.ServiceBuilder = desc.NewService("EchoService").
	AddContract(
		desc.NewContract().
			SetInput(&EchoRequest{}).
			SetOutput(&EchoResponse{}).
			AddRoute(desc.Route("", fastws.RPC("echo"))).
			AddRoute(desc.Route("", fasthttp.GET("/echo/{xp}"))).
			SetHandler(
				contextMW(10*time.Second),
				func(ctx *kit.Context) {
					req, _ := ctx.In().GetMsg().(*EchoRequest)

					ctx.Out().
						SetMsg(
							&EchoResponse{
								Embedded: req.Embedded,
								Output:   req.Input,
							},
						).
						Send()
				}),
	)
