package testenv

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ronykit/testenv/services"

	"github.com/clubpay/ronykit/kit/utils"

	"github.com/clubpay/ronykit/std/gateways/fastws"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/stub"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/fx"
)

func TestFastWS(t *testing.T) {
	Convey("Kit with FastWS", t, func(c C) {
		testCases := map[string]func(t *testing.T, opt fx.Option) func(c C){
			"Edger Server With Huge Websocket Payload": fastwsWithHugePayload,
		}
		for title, fn := range testCases {
			Convey(title,
				fn(
					t, invokeEdgeServerWithFastWS(8082, services.EchoService),
				),
			)
		}
	})
}

func invokeEdgeServerWithFastWS(port int, desc ...kit.ServiceDescriptor) fx.Option {
	return fx.Invoke(
		func(lc fx.Lifecycle) {
			edge := kit.NewServer(
				kit.WithLogger(&stdLogger{}),
				kit.WithErrorHandler(
					func(ctx *kit.Context, err error) {
						fmt.Println("EdgeError: ", err)
					},
				),
				kit.WithGateway(
					fastws.MustNew(
						fastws.WithPredicateKey("cmd"),
						fastws.Listen(fmt.Sprintf("tcp4://0.0.0.0:%d", port)),
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

func fastwsWithHugePayload(t *testing.T, opt fx.Option) func(c C) {
	ctx := context.Background()

	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
			),
		)

		time.Sleep(time.Second * 5)

		wsCtx := stub.New("localhost:8082").
			Websocket(
				stub.WithPredicateKey("cmd"),
			)
		c.So(wsCtx.Connect(ctx, "/"), ShouldBeNil)

		req := &services.EchoRequest{Input: utils.RandomID(12000)}
		res := &services.EchoResponse{}
		err := wsCtx.BinaryMessage(
			ctx, "echo", req, res,
			func(ctx context.Context, msg kit.Message, hdr stub.Header, err error) {
				c.So(err, ShouldBeNil)
				c.So(msg.(*services.EchoResponse).Output, ShouldEqual, req.Input) //nolint:forcetypeassert
			},
		)
		c.So(err, ShouldBeNil)
	}
}
