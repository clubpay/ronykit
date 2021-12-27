package main

import (
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/log"
	"github.com/ronaksoft/ronykit/std/bundle/jsonrpc"
	"syscall"
)

func main() {
	bundle := jsonrpc.New(
		jsonrpc.Listen("tcp4://0.0.0.0:80"),
	)

	bundle.SetHandler(
		"echo",
		func() ronykit.Message {
			return &echoRequest{}
		},
		func(ctx *ronykit.Context) ronykit.Handler {
			req, ok := ctx.Receive().(*echoRequest)
			if !ok {
				ctx.Send(
					&errorMessage{
						Code:    "E01",
						Message: "Request was not echoRequest",
					},
				)

				return nil
			}

			res := &echoResponse{
				RandomID: req.RandomID,
			}

			ctx.Send(res)

			return nil
		},
	)

	// Create, start and wait for shutdown signal of the server.
	ronykit.NewServer(
		ronykit.WithLogger(log.DefaultLogger),
		ronykit.RegisterBundle(bundle),
	).
		Start().
		Shutdown(syscall.SIGHUP)
}
