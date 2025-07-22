package codecserver

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"go.temporal.io/sdk/converter"
)

var _ desc.ServiceDesc = (*Service)(nil)

type Service struct{}

func (s Service) Desc() *desc.Service {
	converter.NewPayloadCodecHTTPHandler()
	return desc.NewService("temporal-codec-server").
		AddContract(
			desc.NewContract().
				AddRoute(desc.Route("Encode", fasthttp.POST("/encode1"))).
				In(kit.RawMessage{}).
				Out(kit.RawMessage{}),
		)
}
