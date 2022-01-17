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
					req, ok := ctx.In().GetMsg().(*echoRequest)

					if !ok {
						ctx.Out().
							SetMsg(
								rest.Err("E01", "Request was not echoRequest"),
							).Send()

						return nil
					}

					ctx.Out().
						SetHdr("Content-Type", "application/json").
						SetMsg(
							&echoResponse{
								RandomID: req.RandomID,
							},
						).Send()

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
					req, ok := ctx.In().GetMsg().(*sumRequest)
					if !ok {
						ctx.Out().
							SetMsg(rest.Err("E01", "Request was not echoRequest")).
							Send()

						return nil
					}

					ctx.Out().
						SetHdr("Content-Type", "application/json").
						SetMsg(
							&sumResponse{
								Val: req.Val1 + req.Val2,
							},
						).Send()

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
					req, ok := ctx.In().GetMsg().(*echoRequest)
					if !ok {
						ctx.Out().
							SetMsg(rest.Err("E01", "Request was not echoRequest")).
							Send()

						return nil
					}

					ctx.Out().
						SetHdr("Content-Type", "application/json").
						SetMsg(
							&echoResponse{
								RandomID: req.RandomID,
							},
						).
						Send()

					return nil
				},
			),
	)
