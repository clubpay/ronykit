package main

import (
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/exmples/simple-jsonrpc/msg"
	"github.com/ronaksoft/ronykit/std/bundle/jsonrpc"
	"time"
)

var sampleService = ronykit.NewService("sample").
	AddRoute(
		map[string]interface{}{
			jsonrpc.QueryPredicate: "echoRequest",
		},
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
	)
