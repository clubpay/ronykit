package main

import (
	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/bundle/rest/mux"
	"github.com/ronaksoft/ronykit/utils"
)

var sampleService = ronykit.
	NewService("sample").
	AddContract(
		ronykit.NewContract().
			SetRouteInfo(
				rest.NewRouteData(
					rest.MethodGet, "/echo/:randomID",
					func(bag mux.Params, data []byte) ronykit.Message {
						m := &echoRequest{}
						m.RandomID = utils.StrToInt64(bag.ByName("randomID"))

						return m
					},
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
			SetRouteInfo(
				rest.NewRouteData(
					rest.MethodGet, "/sum/:val1/:val2",
					func(bag mux.Params, data []byte) ronykit.Message {
						m := &sumRequest{
							Val1: utils.StrToInt64(bag.ByName("val1")),
							Val2: utils.StrToInt64(bag.ByName("val2")),
						}

						return m
					},
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
			SetRouteInfo(
				rest.NewRouteData(
					rest.MethodPost, "/echo",
					func(bag mux.Params, data []byte) ronykit.Message {
						m := &echoRequest{}
						err := json.Unmarshal(data, m)
						if err != nil {
							return nil
						}

						return m
					},
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
