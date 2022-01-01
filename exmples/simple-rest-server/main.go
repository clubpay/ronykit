package main

import (
	"syscall"

	log "github.com/ronaksoft/golog"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/mw"
)

func main() {
	// Create a REST bundle capable of handling REST requests.
	restBundle, err := rest.New(
		rest.Listen(":80"),
	)
	if err != nil {
		panic(err)
	}

	// Create, start and wait for shutdown signal of the server.
	ronykit.NewServer(
		ronykit.WithLogger(log.DefaultLogger),
		ronykit.RegisterBundle(restBundle),
		ronykit.RegisterService(
			ronykit.WrapService(sampleService,
				mw.OpenTelemetry("sample-rest-server"),
			),
		),
	).
		Start().
		Shutdown(syscall.SIGHUP)
}
