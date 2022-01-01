package main

import (
	"fmt"
	"syscall"

	log "github.com/ronaksoft/golog"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/mw/tracekit"
)

func main() {
	// Create, start and wait for shutdown signal of the server.
	defer ronykit.NewServer(
		ronykit.WithLogger(log.DefaultLogger),
		ronykit.RegisterBundle(
			rest.MustNew(
				rest.Listen(":80"),
			),
		),
		ronykit.RegisterService(
			ronykit.WrapService(sampleService,
				tracekit.Trace("sample-rest-server"),
			),
		),
	).
		Start().
		Shutdown(syscall.SIGHUP)

	fmt.Println("Server started.")
}
