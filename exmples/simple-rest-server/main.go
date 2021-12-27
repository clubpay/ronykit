package main

import (
	"syscall"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/log"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/bundle/rest/mux"
	"github.com/ronaksoft/ronykit/utils"
	"github.com/valyala/fasthttp"
)

func main() {
	// Create a REST bundle capable of handling REST requests.
	restBundle, err := rest.New("0.0.0.0:80")
	if err != nil {
		panic(err)
	}

	// Implement Echo API
	restBundle.Set(
		fasthttp.MethodGet, "/echo/:randomID",
		func(bag mux.Params, data []byte) ronykit.Message {
			m := &echoRequest{}
			m.RandomID = utils.StrToInt64(bag.ByName("randomID"))
			return m
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
		ronykit.RegisterBundle(restBundle),
	).
		Start().
		Shutdown(syscall.SIGHUP)
}
