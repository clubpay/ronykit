package main

import (
	"fmt"
	"reflect"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/desc"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
)

type Sample struct {
	desc.Service
}

func NewSample() *Sample {
	s := &Sample{}
	s.Name = "SampleService"
	s.Add(
		*(&desc.Contract{}).
			SetInput(&echoRequest{}).
			AddREST(desc.REST{
				Method: rest.MethodGet,
				Path:   "/echo/:randomID",
			}).
			WithPredicate("echoRequest").
			WithHandlers(echoHandler),
	)

	s.Add(
		*(&desc.Contract{}).
			SetInput(&echoRequest{}).
			AddREST(desc.REST{
				Method: rest.MethodGet,
				Path:   "/sum/:val1/:val2",
			}).
			AddREST(desc.REST{
				Method: rest.MethodPost,
				Path:   "/sum",
			}).
			WithHandlers(sumHandler),
	)

	return s
}

func echoHandler(ctx *ronykit.Context) {
	req, ok := ctx.In().GetMsg().(*echoRequest)
	if !ok {
		ctx.Out().
			SetMsg(
				rest.Err("E01", fmt.Sprintf("Request was not echoRequest: %s", reflect.TypeOf(ctx.In().GetMsg()))),
			).Send()

		return
	}

	ctx.Out().
		SetHdr("Content-Type", "application/json").
		SetMsg(
			&echoResponse{
				RandomID: req.RandomID,
			},
		).Send()

	return
}

func sumHandler(ctx *ronykit.Context) {
	req, ok := ctx.In().GetMsg().(*sumRequest)
	if !ok {
		ctx.Out().
			SetMsg(rest.Err("E01", "Request was not echoRequest")).
			Send()

		return
	}

	ctx.Out().
		SetHdr("Content-Type", "application/json").
		SetMsg(
			&sumResponse{
				Val: req.Val1 + req.Val2,
			},
		).Send()

	return
}
