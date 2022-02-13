package main

import (
	"fmt"
	"os"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/fasthttp"
	"github.com/ronaksoft/ronykit/std/bundle/fastws"
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
				fastws.Listen("tcp4://0.0.0.0:7080"),
				fastws.PredicateKey("cmd"),
			),
			fasthttp.MustNew(
				fasthttp.Listen(":7070"),
			),
		),
		ronykit.RegisterService(NewSample().Desc().Generate()),
	).
		Start().
		Shutdown(os.Kill, os.Interrupt)

	fmt.Println("Server started.")
}
