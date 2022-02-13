package main

import (
	"fmt"
	"runtime"
	"syscall"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/fasthttp"
)

func main() {
	runtime.GOMAXPROCS(4)

	// Create, start and wait for shutdown signal of the server.
	defer ronykit.NewServer(
		ronykit.RegisterBundle(
			fasthttp.MustNew(
				fasthttp.Listen(":80"),
				fasthttp.WithServerName("RonyKIT Server"),
				fasthttp.WithCORS(fasthttp.CORSConfig{}),
			),
		),
		ronykit.RegisterService(
			NewSample().Desc().Generate(),
		),
	).
		Start().
		Shutdown(syscall.SIGHUP)

	fmt.Println("Server started.")
}
