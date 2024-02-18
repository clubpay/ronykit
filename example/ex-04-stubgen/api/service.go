package api

import (
	"fmt"
	"net/http"

	"github.com/clubpay/ronykit/example/ex-04-stubgen/dto"
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
				SetInput(&dto.VeryComplexRequest{}).
				SetOutput(&dto.VeryComplexResponse{}).
				NamedSelector("ComplexDummy", fasthttp.REST(http.MethodPost, "/complexDummy")).
				NamedSelector("ComplexDummy2", fasthttp.REST(http.MethodPost, "/complexDummy/:key1")).
				AddModifier(func(envelope *kit.Envelope) {
					envelope.SetHdr("X-Custom-Header", "justForTestingModifier")
				}).
				SetHandler(DummyHandler),
		)
}

func DummyHandler(ctx *kit.Context) {
	//nolint:forcetypeassert
	req := ctx.In().GetMsg().(*dto.VeryComplexRequest)

	fmt.Println(req.Key1)
	ctx.SetStatusCode(http.StatusOK)
	ctx.In().Reply().
		SetHdr("Content-Type", "application/json").
		SetMsg(
			&dto.VeryComplexResponse{
				Key1:    req.Key1,
				Key1Ptr: req.Key1Ptr,
				MapKey1: req.MapKey1,
			},
		).Send()
}
