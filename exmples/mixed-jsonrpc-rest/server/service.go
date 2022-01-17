package main

import (
	"time"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/exmples/mixed-jsonrpc-rest/msg"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/bundle/rpc"
)

var sampleService = ronykit.NewService("sample").
	AddContract(
		ronykit.NewContract().
			AddRoute(
				rest.GetWithFactory("/echo/:randomID",
					func() interface{} {
						return &msg.EchoRequest{}
					},
				),
				rpc.Route("echoRequest",
					func() ronykit.Message {
						return &msg.EchoRequest{}
					},
				),
			).
			SetHandler(
				func(ctx *ronykit.Context) ronykit.Handler {
					req, ok := ctx.Receive().GetMsg().(*msg.EchoRequest)
					res := ronykit.NewEnvelope()
					if !ok {
						res.SetMsg(rpc.Err("E01", "Request was not echoRequest"))
						ctx.Send(res)

						return nil
					}

					res.SetHdr("ServerTime", time.Now().String())
					res.SetMsg(
						&msg.EchoResponse{
							RandomID: req.RandomID,
						},
					)
					ctx.Send(res)

					return nil
				},
			),
	)
