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
				rpc.Selector("echoRequest",
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

						ctx.Send(
							res.SetMsg(
								rpc.Err("E01", "Request was not echoRequest"),
							),
						)

						return nil
					}

					ctx.Send(
						res.
							SetHdr("ServerTime", time.Now().String()).
							SetMsg(
								&msg.EchoResponse{
									RandomID: req.RandomID,
								},
							),
					)

					return nil
				},
			),
	)
