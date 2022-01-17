package main

import (
	"fmt"
	"os"

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
		ronykit.WithErrorHandler(
			func(ctx *ronykit.Context, err error) {
				fmt.Println("got error: ", err)
			},
		),
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
		Shutdown(os.Kill, os.Interrupt)

	fmt.Println("Server started.")
}
