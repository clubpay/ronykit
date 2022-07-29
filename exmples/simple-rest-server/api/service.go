package api

import (
	"fmt"
	"net/http"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/desc"
	"github.com/clubpay/ronykit/exmples/simple-rest-server/dto"
	"github.com/clubpay/ronykit/std/gateway/fasthttp"
)

var SampleDesc desc.ServiceDescFunc = func() *desc.Service {
	return desc.NewService("SampleService").
		SetEncoding(ronykit.JSON).
		AddError(dto.Err(http.StatusBadRequest, "INPUT")).
		AddContract(
			desc.NewContract().
				SetInput(&dto.EchoRequest{}).
				SetOutput(&dto.EchoResponse{}).
				NamedSelector("EchoGET", fasthttp.REST(http.MethodGet, "/echo/:randomID")).
				NamedSelector("EchoPOST", fasthttp.REST(http.MethodPost, "/echo-post")).
				NamedSelector("EchoRPC", fasthttp.RPC("echoRequest")).
				AddModifier(func(envelope ronykit.Envelope) {
					envelope.SetHdr("X-Custom-Header", "justForTestingModifier")
				}).
				SetHandler(EchoHandler),
		).
		AddContract(
			desc.NewContract().
				SetName("Sum").
				SetInput(&dto.SumRequest{}).
				SetOutput(&dto.SumResponse{}).
				NamedSelector("Sum1", fasthttp.REST(http.MethodGet, "/sum/:val1/:val2")).
				NamedSelector("Sum2", fasthttp.REST(http.MethodPost, "/sum")).
				SetHandler(SumHandler),
		).
		AddContract(
			desc.NewContract().
				SetInput(&dto.SumRequest{}).
				SetOutput(&dto.SumResponse{}).
				NamedSelector("SumRedirect", fasthttp.REST(http.MethodGet, "/sum-redirect/:val1/:val2")).
				Selector(fasthttp.REST(http.MethodPost, "/sum-redirect")).
				SetHandler(SumRedirectHandler),
		).
		AddContract(
			desc.NewContract().
				SetInput(&dto.RedirectRequest{}).
				Selector(fasthttp.REST(http.MethodGet, "/redirect")).
				SetHandler(Redirect),
		)
}

func EchoHandler(ctx *ronykit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(*dto.EchoRequest)

	ctx.Out().
		SetHdr("Content-Type", "application/json").
		SetMsg(
			&dto.EchoResponse{
				RandomID: req.RandomID,
				Ok:       req.Ok,
			},
		).Send()

	return
}

func SumHandler(ctx *ronykit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(*dto.SumRequest)

	ctx.Out().
		SetHdr("Content-Type", "application/json").
		SetMsg(
			&dto.SumResponse{
				EmbeddedHeader: req.EmbeddedHeader,
				Val:            req.Val1 + req.Val2,
			},
		).Send()

	return
}

func SumRedirectHandler(ctx *ronykit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(*dto.SumRequest)

	rc, ok := ctx.Conn().(ronykit.RESTConn)
	if !ok {
		ctx.Out().
			SetMsg(dto.Err(http.StatusBadRequest, "Only supports REST requests")).
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
			SetMsg(dto.Err(http.StatusBadRequest, "Unsupported method")).
			Send()

		return
	}

	return
}

func Redirect(ctx *ronykit.Context) {
	req := ctx.In().GetMsg().(*dto.RedirectRequest) //nolint:forcetypeassert

	rc := ctx.Conn().(ronykit.RESTConn) //nolint:forcetypeassert
	rc.Redirect(http.StatusTemporaryRedirect, req.URL)
}
