package testenv

import (
	"context"
	"testing"
	"time"

	"ronykit/testenv/services"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/stub"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/fx"
)

func TestFastWS(t *testing.T) {
	Convey("Kit with FastWS", t, func(c C) {
		testCases := map[string]func(t *testing.T, opt fx.Option) func(c C){
			"Edge Server With Huge Websocket Payload": fastwsWithHugePayload,
			"Edge Server With Ping and Small Payload": fastwsWithPingAndSmallPayload,
			"Edge Server With Ping Only":              fastwsWithPingOnly,
			"Edge Server With Close":                  fastwsWithClose,
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

func fastwsWithHugePayload(t *testing.T, opt fx.Option) func(c C) {
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
		c.So(wsCtx.Connect(ctx, "/"), ShouldBeNil)

		for i := 0; i < 10; i++ {
			req := &services.EchoRequest{Input: utils.RandomID(10000)}
			res := &services.EchoResponse{}
			err := wsCtx.BinaryMessage(
				ctx, "echo", req, res,
				func(ctx context.Context, msg kit.Message, hdr stub.Header, err error) {
					c.So(err, ShouldBeNil)
					c.So(msg.(*services.EchoResponse).Output, ShouldEqual, req.Input) //nolint:forcetypeassert
				},
			)
			c.So(err, ShouldBeNil)
			time.Sleep(time.Second * 2)
		}
	}
}

func fastwsWithPingAndSmallPayload(t *testing.T, opt fx.Option) func(c C) {
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
		c.So(wsCtx.Connect(ctx, "/"), ShouldBeNil)

		for i := 0; i < 10; i++ {
			req := &services.EchoRequest{Input: utils.RandomID(32)}
			res := &services.EchoResponse{}
			err := wsCtx.BinaryMessage(
				ctx, "echo", req, res,
				func(ctx context.Context, msg kit.Message, hdr stub.Header, err error) {
					c.So(err, ShouldBeNil)
					c.So(msg.(*services.EchoResponse).Output, ShouldEqual, req.Input) //nolint:forcetypeassert
				},
			)
			c.So(err, ShouldBeNil)
			time.Sleep(time.Second)
		}
	}
}

func fastwsWithPingOnly(t *testing.T, opt fx.Option) func(c C) {
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
		c.So(wsCtx.Connect(ctx, "/"), ShouldBeNil)

		_, _ = c.Println("waiting for 10sec ...")
		time.Sleep(time.Second * 10)

	}
}

func fastwsWithClose(t *testing.T, opt fx.Option) func(c C) {
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

		time.Sleep(time.Second * 20)

		c.So(connected, ShouldEqual, 2)
	}
}
