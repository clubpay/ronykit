package main

import (
	"time"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/desc"
	"github.com/ronaksoft/ronykit/exmples/mixed-jsonrpc-rest/msg"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/bundle/rpc"
)

type Sample struct {
	desc.Service
}

func NewSample() *Sample {
	s := &Sample{}
	s.Name = "SampleService"
	s.Add(
		*(&desc.Contract{}).
			SetInput(&msg.EchoRequest{}).
			AddREST(desc.REST{
				Method: rest.MethodGet,
				Path:   "/echo/:randomID",
			}).
			WithPredicate("echoRequest").
			WithHandlers(echoHandler),
	)

	return s
}

func echoHandler(ctx *ronykit.Context) {
	req, ok := ctx.In().GetMsg().(*msg.EchoRequest)
	if !ok {
		ctx.Out().
			SetMsg(
				rpc.Err("E01", "Request was not echoRequest"),
			).
			Send()

		return
	}

	ctx.Out().
		SetHdr("ServerTime", time.Now().String()).
		SetMsg(
			&msg.EchoResponse{
				RandomID: req.RandomID,
			},
		).Send()

	return
}
