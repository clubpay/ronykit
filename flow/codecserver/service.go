package codecserver

import (
	"net/http"
	"path/filepath"

	"github.com/clubpay/ronykit/flow"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ desc.ServiceDesc = (*Service)(nil)

type Service struct {
	routePrefix string
	codec       map[string]converter.PayloadCodec
}

// NewService
// keys: is a map of namespace -> encryptionKey
func NewService(routePrefix string, keys map[string]string) Service {
	svc := Service{
		routePrefix: routePrefix,
		codec:       map[string]converter.PayloadCodec{},
	}

	for namespace, key := range keys {
		svc.codec[namespace] = flow.EncryptedPayloadCodec(key)
	}

	return svc
}

func (s Service) Desc() *desc.Service {
	return desc.NewService("temporal-codec-server").
		AddContract(
			desc.NewContract().
				AddRoute(desc.Route("Encode", fasthttp.POST(filepath.Join(s.routePrefix, "/decode")))).
				In(kit.RawMessage{}).
				Out(kit.RawMessage{}).
				SetHandler(s.Decode),
			desc.NewContract().
				AddRoute(desc.Route("Encode", fasthttp.POST(filepath.Join(s.routePrefix, "/encode")))).
				In(kit.RawMessage{}).
				Out(kit.RawMessage{}).
				SetHandler(s.Decode),
		)
}

func (s *Service) Decode(ctx *kit.Context) {
	msg := ctx.In().GetMsg().(kit.RawMessage)

	var payloadspb commonpb.Payloads

	err := protojson.Unmarshal(msg, &payloadspb)
	if err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		ctx.Out().SetMsg(err.Error()).Send()

		return
	}

	ns := ctx.GetString("X-Namespace", "")

	codec := s.codec[ns]
	if codec == nil {
		codec = s.codec["*"]
		if codec == nil {
			ctx.SetStatusCode(http.StatusBadRequest)
			ctx.Out().SetMsg("codec not found for namespace").Send()

			return
		}
	}

	res, err := codec.Decode(payloadspb.GetPayloads())
	if err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		ctx.Out().SetMsg(err.Error()).Send()

		return
	}

	out, err := protojson.Marshal(&commonpb.Payloads{Payloads: res})
	if err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		ctx.Out().SetMsg(err.Error()).Send()

		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Out().
		SetHdr("Content-Type", "application/json").
		SetMsg(kit.RawMessage(out)).
		Send()
}

func (s *Service) Encode(ctx *kit.Context) {
	msg := ctx.In().GetMsg().(kit.RawMessage)

	var payloadspb commonpb.Payloads

	err := protojson.Unmarshal(msg, &payloadspb)
	if err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		ctx.Out().SetMsg(err.Error()).Send()

		return
	}

	ns := ctx.GetString("X-Namespace", "")

	codec := s.codec[ns]
	if codec == nil {
		codec = s.codec["*"]
		if codec == nil {
			ctx.SetStatusCode(http.StatusBadRequest)
			ctx.Out().SetMsg("codec not found for namespace").Send()

			return
		}
	}

	res, err := codec.Encode(payloadspb.GetPayloads())
	if err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		ctx.Out().SetMsg(err.Error()).Send()

		return
	}

	out, err := protojson.Marshal(&commonpb.Payloads{Payloads: res})
	if err != nil {
		ctx.SetStatusCode(http.StatusBadRequest)
		ctx.Out().SetMsg(err.Error()).Send()

		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Out().
		SetHdr("Content-Type", "application/json").
		SetMsg(kit.RawMessage(out)).
		Send()
}
