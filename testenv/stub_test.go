package testenv

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"testing"

	"ronykit/testenv/services"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/stub"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/fx"
)

func TestDecoder(t *testing.T) {
	Convey("Decoder", t, func(c C) {
		testCases := map[string]func(t *testing.T, opt fx.Option) func(c C){
			"Stub with Run":       stubWithRun,
			"Stub with AutoRun 1": stubWithAutoRun1,
			"Stub With AutoRun 2": stubWithAutoRun2,
		}
		for title, fn := range testCases {
			Convey(title,
				fn(
					t, invokeEdgeServerFastHttp("edge", 8082, services.EchoService),
				),
			)
		}
	})
}

func stubWithRun(t *testing.T, opt fx.Option) func(c C) {
	ctx := context.Background()

	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
			),
		)

		for range 100 {
			X := utils.RandomID(10)
			XP := utils.RandomID(10)
			Y := utils.RandomInt64(100)
			Z := rand.Float64()
			A := utils.S2B(utils.RandomID(10))

			// Set Key to instance 1
			resp := &services.EchoResponse{}
			err := stub.New("localhost:8082").REST().
				SetMethod("GET").
				SetPathF("/echo/%s", XP).
				DefaultResponseHandler(
					func(ctx context.Context, r stub.RESTResponse) *stub.Error {
						c.So(r.StatusCode(), ShouldEqual, http.StatusOK)

						return stub.WrapError(json.Unmarshal(r.GetBody(), resp))
					},
				).
				SetQueryMap(map[string]string{
					"x": X,
					"y": utils.Int64ToStr(Y),
					"z": utils.Float64ToStr(Z),
					"a": utils.B2S(A),
				}).
				Run(ctx).
				Error()
			c.So(err, ShouldBeNil)
			c.So(resp.X, ShouldEqual, X)
			c.So(resp.XP, ShouldEqual, XP)
			c.So(resp.Y, ShouldEqual, Y)
			c.So(resp.Z, ShouldEqual, Z)
			c.So(resp.A, ShouldEqual, A)
		}
	}
}

func stubWithAutoRun1(t *testing.T, opt fx.Option) func(c C) {
	ctx := context.Background()

	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
			),
		)

		for range 1 {
			// Set Key to instance 1
			req := &services.EchoRequest{
				Embedded: services.Embedded{
					X:  utils.RandomID(10),
					XP: utils.RandomID(10),
					Y:  rand.Int63(),
					Z:  rand.Float64(),
					A:  utils.S2B(utils.RandomID(10)),
				},
			}
			resp := &services.EchoResponse{}
			err := stub.New("localhost:8082").REST().
				SetMethod("GET").
				DefaultResponseHandler(
					func(ctx context.Context, r stub.RESTResponse) *stub.Error {
						c.So(r.StatusCode(), ShouldEqual, http.StatusOK)

						return stub.WrapError(json.Unmarshal(r.GetBody(), resp))
					},
				).
				AutoRun(ctx, fmt.Sprintf("/echo/%s", req.XP), kit.JSON, req).
				Error()
			c.So(err, ShouldBeNil)
			c.So(resp.X, ShouldEqual, req.X)
			c.So(resp.XP, ShouldEqual, req.XP)
			c.So(resp.Y, ShouldEqual, req.Y)
			c.So(resp.Z, ShouldEqual, req.Z)
			fmt.Println(string(resp.A), string(req.A))
			c.So(resp.A, ShouldEqual, req.A)
		}
	}
}

func stubWithAutoRun2(t *testing.T, opt fx.Option) func(c C) {
	ctx := context.Background()

	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
			),
		)

		for range 1 {
			// Set Key to instance 1
			req := &services.EchoRequest{
				Embedded: services.Embedded{
					X:  utils.RandomID(10),
					XP: utils.RandomID(10),
					Y:  rand.Int63(),
					Z:  rand.Float64(),
					A:  utils.S2B(utils.RandomID(10)),
				},
			}
			resp := &services.EchoResponse{}
			err := stub.New("localhost:8082").REST().
				SetMethod("GET").
				DefaultResponseHandler(
					func(ctx context.Context, r stub.RESTResponse) *stub.Error {
						c.So(r.StatusCode(), ShouldEqual, http.StatusOK)

						return stub.WrapError(json.Unmarshal(r.GetBody(), resp))
					},
				).
				AutoRun(ctx, "/echo/{xpx}", kit.JSON, req).
				Error()
			c.So(err, ShouldBeNil)
			c.So(resp.X, ShouldEqual, req.X)
			c.So(resp.XP, ShouldEqual, "_")
			c.So(resp.Y, ShouldEqual, req.Y)
			c.So(resp.Z, ShouldEqual, req.Z)
			fmt.Println(string(resp.A), string(req.A))
			c.So(resp.A, ShouldEqual, req.A)
		}
	}
}

func TestWebsocket(t *testing.T) {
	Convey("Websocket", t, func(c C) {
		testCases := map[string]func(t *testing.T, opt fx.Option) func(c C){
			"Websocket Stub [Connect, Reconnect, Disconnect]": stubWebsocket,
		}
		for title, fn := range testCases {
			Convey(title+"FastHTTP",
				fn(t, invokeEdgeServerFastHttp("edge", 8082, services.EchoService)),
			)
			Convey(title+":FastWS",
				fn(t, invokeEdgeServerWithFastWS(8082, services.EchoService)),
			)
		}
	})
}

func stubWebsocket(t *testing.T, opt fx.Option) func(c C) {
	ctx := context.Background()

	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
			),
		)

		for range 200 {
			X := utils.RandomID(10)
			XP := utils.RandomID(10)

			// Set Key to instance 1
			resp := &services.EchoResponse{}
			wsCtx := stub.New("127.0.0.1:8082").
				Websocket(
					stub.WithPredicateKey("cmd"),
				)

			err := wsCtx.Connect(ctx, "/agent/ws")
			c.So(err, ShouldBeNil)

			err = wsCtx.Reconnect(ctx)
			c.So(err, ShouldBeNil)

			err = wsCtx.TextMessage(
				ctx, "echo",
				&services.EchoRequest{
					Embedded: services.Embedded{
						X: X,
					},
					Input: XP,
				},
				resp,
				func(ctx context.Context, msg kit.Message, hdr stub.Header, err error) {
					c.So(err, ShouldBeNil)
					c.So(resp.X, ShouldEqual, X)
					c.So(resp.XP, ShouldEqual, XP)
				},
			)
			c.So(err, ShouldBeNil)

			wsCtx.Disconnect()
		}
	}
}

func TestHttp(t *testing.T) {
	Convey("HTTP", t, func(c C) {
		testCases := map[string]func(t *testing.T, opt fx.Option) func(c C){
			"Compressed": stubHttpCompressed,
		}
		for title, fn := range testCases {
			Convey(title+"FastHTTP",
				fn(t, invokeEdgeServerFastHttp("edge", 8082, services.EchoService)),
			)
		}
	})
}

func stubHttpCompressed(t *testing.T, opt fx.Option) func(c C) {
	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
			),
		)

		for range 200 {
			X := utils.RandomID(10)
			XP := utils.RandomID(10)
			s := stub.New("127.0.0.1:8082")

			err := s.REST().
				SetDeflateBody(
					utils.Ok(json.Marshal(&services.EchoRequest{
						Embedded: services.Embedded{
							X: X,
						},
						Input: XP,
					})),
				).
				DefaultResponseHandler(func(ctx context.Context, r stub.RESTResponse) *stub.Error {
					resp := &services.EchoResponse{}
					err := json.Unmarshal(r.GetBody(), resp)
					c.So(err, ShouldBeNil)
					c.So(resp.X, ShouldEqual, X)
					c.So(resp.XP, ShouldEqual, XP)

					return nil
				}).
				Error()
			c.So(err, ShouldBeNil)
		}
	}
}
