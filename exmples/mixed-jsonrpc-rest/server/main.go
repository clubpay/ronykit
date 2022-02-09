package main

import (
	"fmt"
	"os"

	log "github.com/ronaksoft/golog"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/bundle/rpc"
)

func main() {
	// Create, start and wait for shutdown signal of the server.
	defer ronykit.NewServer(
		ronykit.WithLogger(log.DefaultLogger),
		ronykit.WithErrorHandler(
			func(ctx *ronykit.Context, err error) {
				fmt.Println("got error: ", err)
			},
		),
		ronykit.RegisterBundle(
			rpc.New(
				rpc.Listen("tcp4://0.0.0.0:7080"),
				rpc.PredicateKey("cmd"),
			),
			rest.MustNew(
				rest.Listen(":7070"),
			),
		),
		ronykit.RegisterService(NewSample().Desc().Generate()),
	).
		Start().
		Shutdown(os.Kill, os.Interrupt)

	fmt.Println("Server started.")
}
