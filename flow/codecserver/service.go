package codecserver

import (
	"net/http"
	"path"

	"github.com/clubpay/ronykit/flow"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	headerNamespace = "X-Namespace"

	headerCORSAllowOrigin  = "Access-Control-Allow-Origin"
	headerCORSAllowMethods = "Access-Control-Allow-Methods"
	headerCORSAllowHeaders = "Access-Control-Allow-Headers"
	headerCORSExposeHdrs   = "Access-Control-Expose-Headers"

	corsAllowMethods = "POST, OPTIONS"
	corsAllowHeaders = "Content-Type, X-Namespace, Authorization, X-Requested-With"
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

func (s Service) route(p string) string {
	return path.Join(s.routePrefix, p)
}

func (s Service) Desc() *desc.Service {
	return desc.NewService("temporal-codec-server").
		AddContract(
			desc.NewContract().
				SetName("Decode").
				AddRoute(desc.Route("Decode", fasthttp.POST(s.route("/decode")))).
				SetInputHeader(desc.OptionalHeader(headerNamespace)).
				In(kit.RawMessage{}).
				Out(kit.RawMessage{}).
				SetHandler(s.Decode),
			desc.NewContract().
				SetName("Encode").
				AddRoute(desc.Route("Encode", fasthttp.POST(s.route("/encode")))).
				SetInputHeader(desc.OptionalHeader(headerNamespace)).
				In(kit.RawMessage{}).
				Out(kit.RawMessage{}).
				SetHandler(s.Encode),
			desc.NewContract().
				SetName("DecodeCORS").
				AddRoute(desc.Route("DecodeCORS", fasthttp.REST(http.MethodOptions, s.route("/decode")))).
				In(kit.RawMessage{}).
				Out(kit.RawMessage{}).
				SetHandler(s.CORS),
			desc.NewContract().
				SetName("EncodeCORS").
				AddRoute(desc.Route("EncodeCORS", fasthttp.REST(http.MethodOptions, s.route("/encode")))).
				In(kit.RawMessage{}).
				Out(kit.RawMessage{}).
				SetHandler(s.CORS),
		)
}

func applyCORS(ctx *kit.Context) {
	ctx.PresetHdr(headerCORSAllowOrigin, "*")
	ctx.PresetHdr(headerCORSAllowMethods, corsAllowMethods)
	ctx.PresetHdr(headerCORSAllowHeaders, corsAllowHeaders)
	ctx.PresetHdr(headerCORSExposeHdrs, "Content-Type")
}

func (s *Service) CORS(ctx *kit.Context) {
	applyCORS(ctx)
	ctx.SetStatusCode(http.StatusNoContent)
	ctx.Out().SetMsg(kit.RawMessage{}).Send()
}

func (s *Service) codecFor(ctx *kit.Context) (converter.PayloadCodec, bool) {
	ns := ctx.In().GetHdr(headerNamespace)
	if codec := s.codec[ns]; codec != nil {
		return codec, true
	}

	if codec := s.codec["*"]; codec != nil {
		return codec, true
	}

	return nil, false
}

func (s *Service) Decode(ctx *kit.Context) {
	s.handle(ctx, false)
}

func (s *Service) Encode(ctx *kit.Context) {
	s.handle(ctx, true)
}

func (s *Service) handle(ctx *kit.Context, encode bool) {
	applyCORS(ctx)

	msg, ok := ctx.In().GetMsg().(kit.RawMessage)
	if !ok {
		writeCodecError(ctx, "invalid request body")

		return
	}

	var payloadspb commonpb.Payloads

	err := protojson.Unmarshal(msg, &payloadspb)
	if err != nil {
		writeCodecError(ctx, err.Error())

		return
	}

	codec, ok := s.codecFor(ctx)
	if !ok {
		writeCodecError(ctx, "codec not found for namespace")

		return
	}

	var res []*commonpb.Payload
	if encode {
		res, err = codec.Encode(payloadspb.GetPayloads())
	} else {
		res, err = codec.Decode(payloadspb.GetPayloads())
	}

	if err != nil {
		writeCodecError(ctx, err.Error())

		return
	}

	out, err := protojson.Marshal(&commonpb.Payloads{Payloads: res})
	if err != nil {
		writeCodecError(ctx, err.Error())

		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Out().
		SetHdr("Content-Type", "application/json").
		SetMsg(kit.RawMessage(out)).
		Send()
}

func writeCodecError(ctx *kit.Context, msg string) {
	ctx.SetStatusCode(http.StatusBadRequest)
	ctx.Out().SetMsg(msg).Send()
}
