package main

import (
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/desc"
	"github.com/ronaksoft/ronykit/exmples/mixed-jsonrpc-rest/msg"
	"github.com/ronaksoft/ronykit/std/bundle/fasthttp"
	"github.com/ronaksoft/ronykit/std/bundle/fastws"
)

type Sample struct{}

func NewSample() *Sample {
	return &Sample{}
}

func (s *Sample) Desc() desc.Service {
	d := desc.Service{
		Name: "SampleService",
	}

	d.Add(
		desc.NewContract().
			SetInput(&msg.EchoRequest{}).
			AddSelector(fasthttp.Selector{
				Method: fasthttp.MethodGet,
				Path:   "/echo/:randomID",
			}).
			AddSelector(fastws.Selector{
				Predicate: "echoRequest",
			}).
			SetHandler(echoHandler),
	)

	return d
}

func echoHandler(ctx *ronykit.Context) {
	req, ok := ctx.In().GetMsg().(*msg.EchoRequest)
	if !ok {
		ctx.Out().
			SetMsg(
				fastws.Err("E01", "Request was not echoRequest"),
			).
			Send()

		return
	}

	ctx.Out().
		//SetHdr("ServerTime", utils.Int64ToStr(utils.NanoTime())).
		SetMsg(
			&msg.EchoResponse{
				RandomID: req.RandomID,
			},
		).Send()

	return
}
