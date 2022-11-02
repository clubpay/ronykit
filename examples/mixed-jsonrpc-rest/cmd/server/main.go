package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateway/fasthttp"
	"github.com/clubpay/ronykit/std/gateway/fastws"
)

func main() {
	runtime.GOMAXPROCS(4)

	go func() {
		_ = http.ListenAndServe(":1234", nil)
	}()

	// Create, start and wait for shutdown signal of the server.
	defer kit.NewServer(
		kit.WithErrorHandler(
			func(ctx *kit.Context, err error) {
				fmt.Println("got error: ", err)
			},
		),
		kit.RegisterGateway(
			fastws.MustNew(
				fastws.Listen("tcp4://0.0.0.0:80"),
				fastws.WithPredicateKey("cmd"),
			),
			fasthttp.MustNew(
				fasthttp.WithWebsocketEndpoint("/ws"),
				fasthttp.WithPredicateKey("cmd"),
				fasthttp.Listen(":81"),
			),
		),
		desc.Register(sample),
	).
		Start(context.TODO()).
		PrintRoutes(os.Stdout).
		Shutdown(context.TODO(), os.Kill, os.Interrupt)

	//nolint:forbidigo
	fmt.Println("Server started.")
}
