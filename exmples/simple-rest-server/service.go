package main

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/desc"
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
		desc.NewContract().
			SetInput(&echoRequest{}).
			AddSelector(rest.Selector{
				Method: rest.MethodGet,
				Path:   "/echo/:randomID",
			}).
			AddSelector(rpc.Selector{
				Predicate: "echoRequest",
			}).
			AddModifier(func(envelope *ronykit.Envelope) {
				envelope.SetHdr("X-Custom-Header", "justForTestingModifier")
			}).
			SetHandler(echoHandler),
	)

	s.Add(
		desc.NewContract().
			SetInput(&sumRequest{}).
			AddSelector(rest.Selector{
				Method: rest.MethodGet,
				Path:   "/sum/:val1/:val2",
			}).
			AddSelector(rest.Selector{
				Method: rest.MethodPost,
				Path:   "/sum",
			}).
			SetHandler(sumHandler),
	)

	s.Add(
		desc.NewContract().
			SetInput(&sumRequest{}).
			AddSelector(rest.Selector{
				Method: rest.MethodGet,
				Path:   "/sum-redirect/:val1/:val2",
			}).
			AddSelector(rest.Selector{
				Method: rest.MethodPost,
				Path:   "/sum-redirect",
			}).
			SetHandler(sumRedirectHandler),
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

func sumRedirectHandler(ctx *ronykit.Context) {
	req, ok := ctx.In().GetMsg().(*sumRequest)
	if !ok {
		ctx.Out().
			SetMsg(rest.Err("E01", "Request was not echoRequest")).
			Send()

		return
	}

	rc, ok := ctx.Conn().(ronykit.REST)
	if !ok {
		ctx.Out().
			SetMsg(rest.Err("E01", "Only supports REST requests")).
			Send()

		return
	}

	switch rc.GetMethod() {
	case rest.MethodGet:
		rc.Redirect(
			http.StatusTemporaryRedirect,
			fmt.Sprintf("http://%s/sum/%d/%d", rc.GetHost(), req.Val1, req.Val2),
		)
	case rest.MethodPost:
		rc.Redirect(
			http.StatusTemporaryRedirect,
			fmt.Sprintf("http://%s/sum", rc.GetHost()),
		)
	default:
		ctx.Out().
			SetMsg(rest.Err("E01", "Unsupported method")).
			Send()

		return
	}

	return
}
