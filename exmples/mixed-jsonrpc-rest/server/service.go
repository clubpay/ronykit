package main

import (
	"time"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/exmples/mixed-jsonrpc-rest/msg"
	"github.com/ronaksoft/ronykit/std/bundle/jsonrpc"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/bundle/rest/mux"
	"github.com/ronaksoft/ronykit/utils"
)

var sampleService = ronykit.NewService("sample").
	AddContract(
		ronykit.NewRoute().
			SetData(
				rest.NewRouteData(
					rest.MethodGet, "/echo/:randomID",
					func(bag mux.Params, data []byte) ronykit.Message {
						req := &msg.EchoRequest{
							RandomID: utils.StrToInt64(bag.ByName("randomID")),
						}

						env, err := jsonrpc.NewEnvelope(0, "echoRequest", req)
						if err != nil {
							return nil
						}

						return env
					},
				),
			).
			SetData(jsonrpc.NewRouteData("echoRequest")).
			SetHandler(
				func(ctx *ronykit.Context) ronykit.Handler {
					in, ok := ctx.Receive().(*jsonrpc.Envelope)
					if !ok {
						ctx.Send(jsonrpc.Err("E01", "Request was not echoRequest"))

						return nil
					}
					req := &msg.EchoRequest{}
					err := in.Unmarshal(req)
					if err != nil {
						ctx.Send(jsonrpc.Err("E02", err.Error()))

						return nil
					}
					res := &msg.EchoResponse{
						RandomID: req.RandomID,
					}

					jsonrpc.SendEnvelope(
						ctx, in.ID, "echoResponse", res,
						jsonrpc.WithHeader("ServerTime", time.Now().String()),
					)

					return nil
				},
			),
	)
