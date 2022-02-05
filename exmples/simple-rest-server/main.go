package main

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"runtime"
	"syscall"

	log "github.com/ronaksoft/golog"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
)

func main() {
	runtime.GOMAXPROCS(4)

	// Create, start and wait for shutdown signal of the server.
	defer ronykit.NewServer(
		ronykit.WithLogger(log.DefaultLogger),
		ronykit.RegisterBundle(
			rest.MustNew(
				rest.Listen(":80"),
				rest.WithHttpServer(&fasthttp.Server{
					Name: "RonyKIT Server",
				}),
			),
		),
		ronykit.RegisterService(
			ronykit.WrapService(
				NewSample().Generate(),
			),
		),
	).
		Start().
		Shutdown(syscall.SIGHUP)

	fmt.Println("Server started.")
}
