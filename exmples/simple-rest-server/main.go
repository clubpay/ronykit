package main

import (
	"fmt"
	"runtime"
	"syscall"

	log "github.com/ronaksoft/golog"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/mw/tracekit"
)

func main() {
	runtime.GOMAXPROCS(4)

	// Create, start and wait for shutdown signal of the server.
	defer ronykit.NewServer(
		ronykit.WithLogger(log.DefaultLogger),
		ronykit.RegisterBundle(
			rest.MustNew(
				rest.Listen(":80"),
			),
		),
		ronykit.RegisterService(
			ronykit.WrapService(
				NewSample().Generate(),
				tracekit.Trace("sample-rest-server"),
			),
		),
	).
		Start().
		Shutdown(syscall.SIGHUP)

	fmt.Println("Server started.")
}
