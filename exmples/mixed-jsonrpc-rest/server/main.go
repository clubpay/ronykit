package main

import (
	"fmt"
	"syscall"

	log "github.com/ronaksoft/golog"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/jsonrpc"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
)

func main() {
	// Create, start and wait for shutdown signal of the server.
	defer ronykit.NewServer(
		ronykit.WithLogger(log.DefaultLogger),
		ronykit.RegisterBundle(
			jsonrpc.New(
				jsonrpc.Listen("tcp4://0.0.0.0:80"),
			),
			rest.MustNew(
				rest.Listen(":81"),
			),
		),
		ronykit.RegisterService(sampleService),
	).
		Start().
		Shutdown(syscall.SIGHUP)

	fmt.Println("Server started.")
}
