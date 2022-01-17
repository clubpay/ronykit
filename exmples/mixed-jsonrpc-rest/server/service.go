package main

import (
	"time"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/exmples/mixed-jsonrpc-rest/msg"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/bundle/rpc"
	"github.com/ronaksoft/ronykit/std/contract"
	"github.com/ronaksoft/ronykit/std/service"
)

var sampleService = service.New("sample").
	AddContract(
		contract.New().
			SetSelector(
				rest.Get("/echo/:randomID").
					WithFactory(
						func() ronykit.Message {
							return &msg.EchoRequest{}
						},
					),
				rpc.Route("echoRequest").
					WithFactory(
						func() ronykit.Message {
							return &msg.EchoRequest{}
						},
					),
			).
			SetHandler(
				func(ctx *ronykit.Context) ronykit.Handler {
					req, ok := ctx.In().GetMsg().(*msg.EchoRequest)
					if !ok {
						ctx.Out().
							SetMsg(
								rpc.Err("E01", "Request was not echoRequest"),
							).
							Send()

						return nil
					}

					ctx.Out().
						SetHdr("ServerTime", time.Now().String()).
						SetMsg(
							&msg.EchoResponse{
								RandomID: req.RandomID,
							},
						).Send()

					return nil
				},
			),
	)
