package services

import (
	"encoding/json"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/std/gateways/fastws"
)

var EchoRawService kit.ServiceBuilder = desc.NewService("EchoRawService").
	AddContract(
		desc.NewContract().
			SetInput(&kit.RawMessage{}).
			SetOutput(&kit.RawMessage{}).
			Selector(fastws.RPC("echo")).
			Selector(fasthttp.GET("/echo")).
			SetHandler(
				contextMW(10*time.Second),
				func(ctx *kit.Context) {
					ctx.Out().
						SetMsg(
							kit.RawMessage(utils.Ok(json.Marshal(&EchoResponse{
								Embedded: Embedded{
									X:  "x",
									XP: "xp",
									Y:  10,
									Z:  11,
									A:  nil,
								},
								Output: "output",
							})),
							)).
						Send()
				}),
	)
