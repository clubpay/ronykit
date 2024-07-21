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
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/stub"
	"github.com/redis/go-redis/v9"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/fx"
)

func TestDecoder(t *testing.T) {
	Convey("Decoder", t, func(c C) {
		testCases := map[string]fx.Option{
			"Test Get with Embedded": fx.Options(
				// fx.Invoke(invokeRedisMonitor),
				invokeEdgeServer("edge", 8082, services.EchoService),
			),
		}
		for title, opts := range testCases {
			Convey(title+" Run", echoWithGetQueryParams1(t, opts))
			Convey(title+" AutoRun", echoWithGetQueryParams2(t, opts))
		}
	})
}

func invokeEdgeServer(_ string, port int, desc ...kit.ServiceBuilder) fx.Option {
	return fx.Invoke(
		func(lc fx.Lifecycle, _ *redis.Client) {
			edge := kit.NewServer(
				kit.WithLogger(common.NewStdLogger()),
				kit.WithErrorHandler(
					func(ctx *kit.Context, err error) {
						fmt.Println("EdgeError: ", err)
					},
				),
				kit.WithGateway(
					fasthttp.MustNew(
						fasthttp.WithDisableHeaderNamesNormalizing(),
						fasthttp.Listen(fmt.Sprintf(":%d", port)),
					),
				),
				kit.WithServiceBuilder(desc...),
			)

			lc.Append(
				fx.Hook{
					OnStart: func(ctx context.Context) error {
						edge.Start(ctx)

						return nil
					},
					OnStop: func(ctx context.Context) error {
						edge.Shutdown(ctx)

						return nil
					},
				},
			)
		},
	)
}

func echoWithGetQueryParams1(t *testing.T, opt fx.Option) func(c C) {
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

func echoWithGetQueryParams2(t *testing.T, opt fx.Option) func(c C) {
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
