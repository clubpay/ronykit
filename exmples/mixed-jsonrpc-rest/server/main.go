package main

import (
	"fmt"
	"syscall"

	"github.com/goccy/go-json"
	log "github.com/ronaksoft/golog"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/bundle/rpc"
)

func main() {
	// Create, start and wait for shutdown signal of the server.
	defer ronykit.NewServer(
		ronykit.WithLogger(log.DefaultLogger),
		ronykit.RegisterBundle(
			rpc.New(
				rpc.Listen("tcp4://0.0.0.0:80"),
				rpc.Decoder(
					func(data []byte, e *rpc.Envelope) error {
						return json.Unmarshal(data, e)
					},
				),
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
