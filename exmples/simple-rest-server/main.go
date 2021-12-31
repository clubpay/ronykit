package main

import (
	"syscall"

	"github.com/goccy/go-json"
	log "github.com/ronaksoft/golog"
	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/std/bundle/rest"
	"github.com/ronaksoft/ronykit/std/bundle/rest/mux"
	"github.com/ronaksoft/ronykit/utils"
	"github.com/valyala/fasthttp"
)

func main() {
	// Create a REST bundle capable of handling REST requests.
	restBundle, err := rest.New(
		rest.Listen(":80"),
	)
	if err != nil {
		panic(err)
	}

	// Implement Echo API
	restBundle.SetHandler(
		fasthttp.MethodGet, "/echo/:randomID",
		func(bag mux.Params, data []byte) ronykit.Message {
			m := &echoRequest{}
			m.RandomID = utils.StrToInt64(bag.ByName("randomID"))

			return m
		},
		func(ctx *ronykit.Context) ronykit.Handler {
			req, ok := ctx.Receive().(*echoRequest)
			if !ok {
				ctx.Send(rest.Err("E01", "Request was not echoRequest"))

				return nil
			}

			res := &echoResponse{
				RandomID: req.RandomID,
			}

			ctx.Send(res)

			return nil
		},
	)

	// Implement Sum API
	restBundle.SetHandler(
		fasthttp.MethodGet, "/sum/:val1/:val2",
		func(bag mux.Params, data []byte) ronykit.Message {
			m := &sumRequest{
				Val1: utils.StrToInt64(bag.ByName("val1")),
				Val2: utils.StrToInt64(bag.ByName("val2")),
			}

			return m
		},
		func(ctx *ronykit.Context) ronykit.Handler {
			req, ok := ctx.Receive().(*sumRequest)
			if !ok {
				ctx.Send(rest.Err("E01", "Request was not echoRequest"))

				return nil
			}

			res := &sumResponse{
				Val: req.Val1 + req.Val2,
			}

			ctx.Send(res)

			return nil
		},
	)

	// Implement Echo with POST request
	restBundle.SetHandler(
		fasthttp.MethodPost, "/echo",
		func(bag mux.Params, data []byte) ronykit.Message {
			m := &echoRequest{}
			err := json.Unmarshal(data, m)
			if err != nil {
				return nil
			}

			return m
		},
		func(ctx *ronykit.Context) ronykit.Handler {
			req, ok := ctx.Receive().(*echoRequest)
			if !ok {
				ctx.Send(rest.Err("E01", "Request was not echoRequest"))

				return nil
			}

			ctx.Set("Content-Type", "application/json")
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
