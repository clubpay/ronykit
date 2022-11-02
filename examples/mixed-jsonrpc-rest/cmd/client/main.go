package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/clubpay/ronykit/examples/internal/testlog"
	"github.com/clubpay/ronykit/examples/simple-rest-server/dto"
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/stub"
)

func main() {
	wsCtx := stub.New("127.0.0.1:80",
		stub.WithDialTimeout(time.Second*3),
		stub.WithLogger(testlog.Log{}),
	).
		Websocket(
			stub.WithPredicateKey("cmd"),
			stub.WithPingTime(5*time.Second),
			stub.WithOnConnectHandler(
				func(wCtx *stub.WebsocketCtx) {
					err := wCtx.TextMessage(
						context.Background(), "echoRequest",
						&dto.EchoRequest{
							RandomID: 2374,
							Ok:       true,
						},
						&dto.EchoResponse{},
						func(ctx context.Context, msg kit.Message, hdr stub.Header, err error) {
							res := msg.(*dto.EchoResponse)
							fmt.Println("Received response:", res.Ok, res.RandomID)
						},
					)
					if err != nil {
						panic(err)
					}

				},
			),
		)

	ctx, cf := context.WithTimeout(context.Background(), time.Second*5)
	defer cf()

	err := wsCtx.Connect(ctx, "/")
	if err != nil {
		panic(err)
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Kill, os.Interrupt)
	<-ch
	_ = wsCtx.Disconnect()
}
