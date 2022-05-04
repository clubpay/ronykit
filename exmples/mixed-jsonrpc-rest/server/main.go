package main

import (
	"fmt"
	"os"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/std/gateway/fasthttp"
	"github.com/clubpay/ronykit/std/gateway/fastws"
)

func main() {
	// Create, start and wait for shutdown signal of the server.
	defer ronykit.NewServer(
		ronykit.WithErrorHandler(
			func(ctx *ronykit.Context, err error) {
				fmt.Println("got error: ", err)
			},
		),
		ronykit.RegisterGateway(
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
		ronykit.RegisterServiceDesc(NewSample().Desc()),
	).
		Start().
		Shutdown(os.Kill, os.Interrupt)

	//nolint:forbidigo
	fmt.Println("Server started.")
}
