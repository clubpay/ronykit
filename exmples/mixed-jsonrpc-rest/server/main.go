package main

import (
	"fmt"
	"os"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/std/bundle/fasthttp"
	"github.com/clubpay/ronykit/std/bundle/fastws"
)

func main() {
	// Create, start and wait for shutdown signal of the server.
	defer ronykit.NewServer(
		ronykit.WithErrorHandler(
			func(ctx *ronykit.Context, err error) {
				fmt.Println("got error: ", err)
			},
		),
		ronykit.RegisterBundle(
			fastws.MustNew(
				fastws.Listen("tcp4://0.0.0.0:80"),
				fastws.WithPredicateKey("cmd"),
			),
			fasthttp.MustNew(
				fasthttp.Listen(":81"),
			),
		),
		ronykit.RegisterService(NewSample().Desc().Generate()),
	).
		Start().
		Shutdown(os.Kill, os.Interrupt)

	//nolint:forbidigo
	fmt.Println("Server started.")
}
