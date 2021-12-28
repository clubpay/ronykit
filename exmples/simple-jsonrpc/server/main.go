package main

import (
	"syscall"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/exmples/simple-jsonrpc/msg"
	"github.com/ronaksoft/ronykit/log"
	"github.com/ronaksoft/ronykit/std/bundle/jsonrpc"
)

func main() {
	bundle := jsonrpc.New(
		jsonrpc.Listen("tcp4://0.0.0.0:80"),
	)

	bundle.SetHandler(
		"echoRequest",
		func(ctx *ronykit.Context) ronykit.Handler {
			in, ok := ctx.Receive().(*jsonrpc.Envelope)
			if !ok {
				ctx.Send(
					&msg.Error{
						Code:    "E01",
						Message: "Request was not echoRequest",
					},
				)

				return nil
			}
			req := &msg.EchoRequest{}
			err := in.Unmarshal(req)
			if err != nil {
				ctx.Send(
					&msg.Error{
						Code:    "E01",
						Message: err.Error(),
					},
				)

				return nil
			}
			res := &msg.EchoResponse{
				RandomID: req.RandomID,
			}

			out := &jsonrpc.Envelope{
				Predicate: "echoResponse",
				ID:        in.ID,
			}
			ctx.Error(out.SetPayload(res))
			ctx.Send(out)

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
