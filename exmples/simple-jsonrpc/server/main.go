package main

import (
	"syscall"

	log "github.com/ronaksoft/golog"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/jsonrpc"
)

func main() {
	bundle := jsonrpc.New(
		jsonrpc.Listen("tcp4://0.0.0.0:80"),
	)

	// Create, start and wait for shutdown signal of the server.
	ronykit.NewServer(
		ronykit.WithLogger(log.DefaultLogger),
		ronykit.RegisterBundle(bundle),
		ronykit.RegisterService(sampleService),
	).
		Start().
		Shutdown(syscall.SIGHUP)
}
