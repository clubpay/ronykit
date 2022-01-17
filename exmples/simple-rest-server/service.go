package main

import (
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
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
		contract.New().
			SetSelector(
				rest.Get("/sum/:val1/:val2").
					WithFactory(
						func() ronykit.Message {
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
		contract.New().
			SetSelector(
				rest.Post("/sum").
					WithFactory(
						func() ronykit.Message {
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
