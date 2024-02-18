package testenv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"ronykit/testenv/services"

	"github.com/clubpay/ronykit/std/clusters/p2pcluster"

	"github.com/redis/go-redis/v9"

	"github.com/clubpay/ronykit/kit/utils"

	"github.com/clubpay/ronykit/std/gateways/fasthttp"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/stub"
	"github.com/clubpay/ronykit/std/clusters/rediscluster"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/fx"
)

func TestKitWithCluster(t *testing.T) {
	Convey("Kit with Cluster", t, func(c C) {
		testScenarios := map[string]func(t *testing.T, opt fx.Option) func(c C){
			//"KeyValue Store - With Redis": kitWithClusterWithRedis,
			"KeyValue Store - With P2P": kitWithClusterWithP2P,
		}
		for scenarioName, f := range testScenarios {
			Convey(
				fmt.Sprintf("%s", scenarioName),
				f(
					t,
					fx.Options(
					// fx.Invoke(invokeRedisMonitor),
					),
				),
			)
		}
	})
}

func kitWithClusterWithRedis(t *testing.T, opt fx.Option) func(c C) {
	ctx := context.Background()

	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
				invokeEdgeServerWithRedis("edge1", 8082, services.SimpleKeyValueService),
				invokeEdgeServerWithRedis("edge2", 8083, services.SimpleKeyValueService),
			),
		)

		time.Sleep(time.Second * 3)

		// Set Key to instance 1
		restCtx := stub.New("localhost:8082").REST()
		resp := &services.KeyValue{}
		err := restCtx.
			SetMethod("POST").
			DefaultResponseHandler(
				func(ctx context.Context, r stub.RESTResponse) *stub.Error {
					c.So(r.StatusCode(), ShouldEqual, http.StatusOK)

					return stub.WrapError(json.Unmarshal(r.GetBody(), resp))
				},
			).
			AutoRun(ctx, "/set-key", kit.JSON, &services.SetRequest{Key: "test", Value: "testValue"}).
			Error()
		c.So(err, ShouldBeNil)
		c.So(resp.Key, ShouldEqual, "test")
		c.So(resp.Value, ShouldEqual, "testValue")

		// Get Key from instance 2
		restCtx = stub.New("localhost:8083").REST()
		err = restCtx.
			SetMethod("GET").
			SetHeader("Conn-Hdr-In", "MyValue").
			SetHeader("Envelope-Hdr-In", "EnvelopeValue").
			DefaultResponseHandler(
				func(ctx context.Context, r stub.RESTResponse) *stub.Error {
					c.So(r.GetHeader("Conn-Hdr-Out"), ShouldEqual, "MyValue")
					c.So(r.GetHeader("Envelope-Hdr-Out"), ShouldEqual, "EnvelopeValue")
					c.So(r.StatusCode(), ShouldEqual, http.StatusOK)

					return stub.WrapError(json.Unmarshal(r.GetBody(), resp))
				},
			).
			AutoRun(ctx, "/get-key/{key}", kit.JSON, &services.GetRequest{Key: "test"}).
			Error()
		c.So(err, ShouldBeNil)
		c.So(resp.Key, ShouldEqual, "test")
		c.So(resp.Value, ShouldEqual, "testValue")
	}
}

func invokeEdgeServerWithRedis(_ string, port int, desc ...kit.ServiceDescriptor) fx.Option {
	return fx.Invoke(
		func(lc fx.Lifecycle, _ *redis.Client) {
			edge := kit.NewServer(
				kit.WithCluster(
					rediscluster.MustNew(
						"testCluster",
						rediscluster.WithRedisClient(utils.Must(getRedis())),
					),
				),
				kit.WithLogger(&stdLogger{}),
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
				kit.WithServiceDesc(desc...),
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

func kitWithClusterWithP2P(t *testing.T, opt fx.Option) func(c C) {
	ctx := context.Background()

	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
				invokeEdgeServerWithP2P("edge1", 8082, services.SimpleKeyValueService),
				invokeEdgeServerWithP2P("edge2", 8083, services.SimpleKeyValueService),
			),
		)

		time.Sleep(time.Second * 15)

		// Set Key to instance 1
		restCtx := stub.New("localhost:8082").REST()
		resp := &services.KeyValue{}
		err := restCtx.
			SetMethod("POST").
			DefaultResponseHandler(
				func(ctx context.Context, r stub.RESTResponse) *stub.Error {
					c.So(r.StatusCode(), ShouldEqual, http.StatusOK)

					return stub.WrapError(json.Unmarshal(r.GetBody(), resp))
				},
			).
			AutoRun(ctx, "/set-key", kit.JSON, &services.SetRequest{Key: "test", Value: "testValue"}).
			Error()
		c.So(err, ShouldBeNil)
		c.So(resp.Key, ShouldEqual, "test")
		c.So(resp.Value, ShouldEqual, "testValue")

		// Get Key from instance 2
		restCtx = stub.New("localhost:8083").REST()
		err = restCtx.
			SetMethod("GET").
			SetHeader("Conn-Hdr-In", "MyValue").
			SetHeader("Envelope-Hdr-In", "EnvelopeValue").
			DefaultResponseHandler(
				func(ctx context.Context, r stub.RESTResponse) *stub.Error {
					c.So(r.GetHeader("Conn-Hdr-Out"), ShouldEqual, "MyValue")
					c.So(r.GetHeader("Envelope-Hdr-Out"), ShouldEqual, "EnvelopeValue")
					c.So(r.StatusCode(), ShouldEqual, http.StatusOK)

					return stub.WrapError(json.Unmarshal(r.GetBody(), resp))
				},
			).
			AutoRun(ctx, "/get-key/{key}", kit.JSON, &services.GetRequest{Key: "test"}).
			Error()
		c.So(err, ShouldBeNil)
		c.So(resp.Key, ShouldEqual, "test")
		c.So(resp.Value, ShouldEqual, "testValue")
	}
}

func invokeEdgeServerWithP2P(_ string, port int, desc ...kit.ServiceDescriptor) fx.Option {
	return fx.Invoke(
		func(lc fx.Lifecycle, _ *redis.Client) {
			edge := kit.NewServer(
				kit.WithCluster(
					p2pcluster.New(
						"testCluster",
						p2pcluster.WithLogger(&stdLogger{}),
						p2pcluster.WithBroadcastInterval(time.Second),
					),
				),
				kit.WithLogger(&stdLogger{}),
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
				kit.WithServiceDesc(desc...),
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
