package main

import (
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
)

var sampleService = ronykit.
	NewService("sample").
	AddContract(
		ronykit.NewContract().
			AddRouteInfo(
				rest.NewRouteData(
					rest.MethodGet, "/echo/:randomID",
					rest.ReflectDecoder(
						func() interface{} {
							return &echoRequest{}
						},
					),
				),
			).
			SetHandler(
				func(ctx *ronykit.Context) ronykit.Handler {
					req, ok := ctx.Receive().(*echoRequest)
					if !ok {
						ctx.Send(rest.Err("E01", "Request was not echoRequest"))

						return nil
					}

					ctx.Set("Content-Type", "application/json")
					res := &echoResponse{
						RandomID: req.RandomID,
					}

					ctx.Send(res, "Content-Type")

					return nil
				},
			),
	).
	AddContract(
		ronykit.NewContract().
			AddRouteInfo(
				rest.NewRouteData(
					rest.MethodGet, "/sum/:val1/:val2",
					rest.ReflectDecoder(
						func() interface{} {
							return &sumRequest{}
						},
					),
				),
			).
			SetHandler(
				func(ctx *ronykit.Context) ronykit.Handler {
					req, ok := ctx.Receive().(*sumRequest)
					if !ok {
						ctx.Send(rest.Err("E01", "Request was not echoRequest"))

						return nil
					}

					res := &sumResponse{
						Val: req.Val1 + req.Val2,
					}

					ctx.Send(res, "Content-Type")

					return nil
				},
			),
	).
	AddContract(
		ronykit.NewContract().
			AddRouteInfo(
				rest.NewRouteData(
					rest.MethodPost, "/echo",
					rest.ReflectDecoder(
						func() interface{} {
							return &sumRequest{}
						},
					),
				),
			).
			SetHandler(
				func(ctx *ronykit.Context) ronykit.Handler {
					req, ok := ctx.Receive().(*echoRequest)
					if !ok {
						ctx.Send(rest.Err("E01", "Request was not echoRequest"))

						return nil
					}

					ctx.Set("Content-Type", "application/json")
					res := &echoResponse{
						RandomID: req.RandomID,
					}

					ctx.Send(res, "Content-Type")

					return nil
				},
			),
	)
