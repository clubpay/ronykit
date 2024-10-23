package testenv

import (
	"context"
	"testing"
	"time"

	"ronykit/testenv/services"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/stub"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/fx"
)

func TestFastHTTP(t *testing.T) {
	Convey("Kit with FastHttp", t, func(c C) {
		testCases := map[string]func(t *testing.T, opt fx.Option) func(c C){
			"Edge Server With Ping Only": fasthttpWithPingOnly,
			"Edge Server With Close":     fasthttpWithClose,
		}
		for title, fn := range testCases {
			Convey(title,
				fn(
					t, invokeEdgeServerFastHttp("", 8082, services.EchoService),
				),
			)
		}
	})
}

func fasthttpWithPingOnly(t *testing.T, opt fx.Option) func(c C) {
	ctx := context.Background()

	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
			),
		)

		time.Sleep(time.Second * 2)

		wsCtx := stub.New(
			"localhost:8082",
			stub.WithLogger(common.NewStdLogger()),
		).
			Websocket(
				stub.WithPredicateKey("cmd"),
				stub.WithPingTime(time.Second),
			)
		c.So(wsCtx.Connect(ctx, "/agent/ws"), ShouldBeNil)

		_, _ = c.Println("waiting for 10sec ...")
		time.Sleep(time.Second * 10)

	}
}

func fasthttpWithClose(t *testing.T, opt fx.Option) func(c C) {
	ctx := context.Background()

	return func(c C) {
		Prepare(
			t, c,
			fx.Options(
				opt,
			),
		)

		time.Sleep(time.Second * 2)

		connected := 0
		wsCtx := stub.New(
			"localhost:8082",
			stub.WithLogger(common.NewStdLogger()),
		).
			Websocket(
				stub.WithPredicateKey("cmd"),
				stub.WithPingTime(time.Second),
				stub.WithOnConnectHandler(
					func(ctx *stub.WebsocketCtx) {
						connected += 1
					}),
			)
		c.So(wsCtx.Connect(ctx, "/agent/ws"), ShouldBeNil)

		_, _ = c.Println("waiting for 3sec ...")
		time.Sleep(time.Second * 3)

		req := &services.CloseRequest{}
		res := &services.CloseResponse{}
		err := wsCtx.BinaryMessage(
			ctx, "close", req, res,
			func(ctx context.Context, msg kit.Message, hdr stub.Header, err error) {
				c.So(err, ShouldBeNil)
				c.So(msg.(*services.CloseResponse).Close, ShouldBeTrue) //nolint:forcetypeassert
			},
		)
		c.So(err, ShouldBeNil)

		time.Sleep(time.Second * 10)

		c.So(connected, ShouldEqual, 2)
	}
}
