package api

import (
	"fmt"
	"net/http"

	"github.com/clubpay/ronykit/example/ex-02-rest/dto"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

var SampleDesc desc.ServiceDescFunc = func() *desc.Service {
	return desc.NewService("SampleService").
		SetEncoding(kit.JSON).
		AddError(dto.Err(http.StatusBadRequest, "INPUT")).
		AddContract(
			desc.NewContract().
				SetInput(&dto.EchoRequest{}).
				SetOutput(&dto.EchoResponse{}).
				NamedSelector("EchoGET", fasthttp.REST(http.MethodGet, "/echo/:randomID")).
				NamedSelector("EchoPOST", fasthttp.REST(http.MethodPost, "/echo-post")).
				NamedSelector("EchoRPC", fasthttp.RPC("echoRequest")).
				AddModifier(func(envelope *kit.Envelope) {
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
		).
		AddContract(
			desc.NewContract().
				SetInput(kit.RawMessage{}).
				SetOutput(kit.RawMessage{}).
				Selector(fasthttp.REST(http.MethodPost, "/raw_echo")).
				Selector(fasthttp.RPC("rawEcho")).
				SetHandler(RawEchoHandler),
		).
		AddContract(
			desc.NewContract().
				SetInput(kit.MultipartFormMessage{}).
				SetOutput(kit.RawMessage{}).
				Selector(fasthttp.REST(http.MethodPost, "/upload")).
				SetHandler(UploadHandler),
		)
}

func EchoHandler(ctx *kit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(*dto.EchoRequest)

	ctx.In().Reply().
		SetHdr("Content-Type", "application/json").
		SetMsg(
			&dto.EchoResponse{
				RandomID: req.RandomID,
				Ok:       req.Ok,
			},
		).Send()
}

func RawEchoHandler(ctx *kit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(kit.RawMessage)

	fmt.Println("RawEchoHandler", string(req))
	ctx.In().Reply().SetMsg(req).Send()
}

func SumHandler(ctx *kit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(*dto.SumRequest)

	ctx.In().Reply().
		SetHdr("Content-Type", "application/json").
		SetMsg(
			&dto.SumResponse{
				EmbeddedHeader: req.EmbeddedHeader,
				Val:            req.Val1 + req.Val2,
			},
		).Send()
}

func SumRedirectHandler(ctx *kit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(*dto.SumRequest)

	rc, ok := ctx.Conn().(kit.RESTConn)
	if !ok {
		ctx.In().Reply().
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
		ctx.In().Reply().
			SetMsg(dto.Err(http.StatusBadRequest, "Unsupported method")).
			Send()
	}
}

func Redirect(ctx *kit.Context) {
	req := ctx.In().GetMsg().(*dto.RedirectRequest) //nolint:forcetypeassert

	rc := ctx.Conn().(kit.RESTConn) //nolint:forcetypeassert
	rc.Redirect(http.StatusTemporaryRedirect, req.URL)
}

func UploadHandler(ctx *kit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(kit.MultipartFormMessage)

	frm := req.GetForm()
	fmt.Println(frm.File)

	ctx.In().Reply().
		SetHdr("Content-Type", "application/json").
		SetMsg(kit.RawMessage{}).
		Send()
}
