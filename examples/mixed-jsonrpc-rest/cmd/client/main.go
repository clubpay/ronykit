package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/examples/simple-rest-server/dto"
	"github.com/clubpay/ronykit/stub"
)

func main() {
	s := stub.New("127.0.0.1:80")
	wsCtx := s.Websocket(
		stub.WithPredicateKey("cmd"),
	)

	err := wsCtx.Connect(context.Background(), "/ws")
	if err != nil {
		panic(err)
	}

	ctx, cf := context.WithTimeout(context.Background(), time.Second*3)
	defer cf()

	wg := sync.WaitGroup{}
	wg.Add(1)
	err = wsCtx.TextMessage(
		ctx, "echoRequest",
		&dto.EchoRequest{
			RandomID: 2374,
			Ok:       true,
		}, &dto.EchoResponse{},
		func(ctx context.Context, msg ronykit.Message, hdr stub.Header, err error) {
			res := msg.(*dto.EchoResponse)
			fmt.Println("Received response:", res.Ok, res.RandomID)
			wg.Done()
		})
	if err != nil {
		panic(err)
	}

	wg.Wait()
}
