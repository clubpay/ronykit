package main

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/desc"
	"github.com/ronaksoft/ronykit/std/bundle/fasthttp"
	"github.com/ronaksoft/ronykit/std/bundle/fastws"
)

type Sample struct{}

func NewSample() *Sample {
	s := &Sample{}

	return s
}

func (x *Sample) Desc() *desc.Service {
	return desc.NewService("SampleService").
		Add(
			desc.NewContract().
				SetInput(&echoRequest{}).
				AddSelector(fasthttp.Selector{
					Method:    fasthttp.MethodGet,
					Predicate: "echo",
					Path:      "/echo/:randomID",
				}).
				AddSelector(fastws.Selector{
					Predicate: "echoRequest",
				}).
				AddModifier(func(envelope *ronykit.Envelope) {
					envelope.SetHdr("X-Custom-Header", "justForTestingModifier")
				}).
				SetHandler(echoHandler),
		).
		Add(
			desc.NewContract().
				SetInput(&sumRequest{}).
				AddSelector(fasthttp.Selector{
					Method: fasthttp.MethodGet,
					Path:   "/sum/:val1/:val2",
				}).
				AddSelector(fasthttp.Selector{
					Method: fasthttp.MethodPost,
					Path:   "/sum",
				}).
				SetHandler(sumHandler),
		).
		Add(
			desc.NewContract().
				SetInput(&sumRequest{}).
				AddSelector(fasthttp.Selector{
					Method: fasthttp.MethodGet,
					Path:   "/sum-redirect/:val1/:val2",
				}).
				AddSelector(fasthttp.Selector{
					Method: fasthttp.MethodPost,
					Path:   "/sum-redirect",
				}).
				SetHandler(sumRedirectHandler),
		)
}

func echoHandler(ctx *ronykit.Context) {
	req, ok := ctx.In().GetMsg().(*echoRequest)
	if !ok {
		ctx.Out().
			SetMsg(
				fasthttp.Err("E01", fmt.Sprintf("Request was not echoRequest: %s", reflect.TypeOf(ctx.In().GetMsg()))),
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
			SetMsg(fasthttp.Err("E01", "Request was not echoRequest")).
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
			SetMsg(fasthttp.Err("E01", "Request was not echoRequest")).
			Send()

		return
	}

	rc, ok := ctx.Conn().(ronykit.RESTConn)
	if !ok {
		ctx.Out().
			SetMsg(fasthttp.Err("E01", "Only supports REST requests")).
			Send()

		return
	}

	switch rc.GetMethod() {
	case fasthttp.MethodGet:
		rc.Redirect(
			http.StatusTemporaryRedirect,
			fmt.Sprintf("http://%s/sum/%d/%d", rc.GetHost(), req.Val1, req.Val2),
		)
	case fasthttp.MethodPost:
		rc.Redirect(
			http.StatusTemporaryRedirect,
			fmt.Sprintf("http://%s/sum", rc.GetHost()),
		)
	default:
		ctx.Out().
			SetMsg(fasthttp.Err("E01", "Unsupported method")).
			Send()

		return
	}

	return
}
