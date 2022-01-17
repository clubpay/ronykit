package main

import (
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
)

var sampleService = ronykit.
	NewService("sample").
	AddContract(
		ronykit.NewContract().
			AddRoute(
				rest.GetWithFactory("/echo/:randomID",
					func() interface{} {
						return &echoRequest{}
					},
				),
			).
			SetHandler(
				func(ctx *ronykit.Context) ronykit.Handler {
					req, ok := ctx.Receive().GetMsg().(*echoRequest)
					res := ronykit.NewEnvelope()

					if !ok {
						res.SetMsg(
							rest.Err("E01", "Request was not echoRequest"),
						)

						ctx.Send(res)

						return nil
					}

					res.SetHdr("Content-Type", "application/json")
					res.SetMsg(
						&echoResponse{
							RandomID: req.RandomID,
						},
					)

					ctx.Send(res)

					return nil
				},
			),
	).
	AddContract(
		ronykit.NewContract().
			AddRoute(
				rest.GetWithFactory("/sum/:val1/:val2",
					func() interface{} {
						return &sumRequest{}
					},
				),
			).
			SetHandler(
				func(ctx *ronykit.Context) ronykit.Handler {
					req, ok := ctx.Receive().GetMsg().(*sumRequest)
					res := ronykit.NewEnvelope()
					if !ok {
						res.SetMsg(rest.Err("E01", "Request was not echoRequest"))
						ctx.Send(res)

						return nil
					}

					res.SetHdr("Content-Type", "application/json")
					res.SetMsg(
						&sumResponse{
							Val: req.Val1 + req.Val2,
						},
					)
					ctx.Send(res)

					return nil
				},
			),
	).
	AddContract(
		ronykit.NewContract().
			AddRoute(
				rest.PostWithFactory("/sum",
					func() interface{} {
						return &sumRequest{}
					},
				),
			).
			SetHandler(
				func(ctx *ronykit.Context) ronykit.Handler {
					req, ok := ctx.Receive().GetMsg().(*echoRequest)
					res := ronykit.NewEnvelope()

					if !ok {
						res.SetMsg(rest.Err("E01", "Request was not echoRequest"))
						ctx.Send(res)

						return nil
					}

					res.SetHdr("Content-Type", "application/json")
					res.SetMsg(
						&echoResponse{
							RandomID: req.RandomID,
						},
					)

					ctx.Send(res)

					return nil
				},
			),
	)
