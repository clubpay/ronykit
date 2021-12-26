package main

import (
	"syscall"

	"github.com/goccy/go-json"

	"github.com/ronaksoft/ronykit"
	"github.com/ronaksoft/ronykit/log"
	"github.com/ronaksoft/ronykit/utils"
	"github.com/valyala/fasthttp"

	"github.com/ronaksoft/ronykit/std/bundle/rest"
	tcpGateway "github.com/ronaksoft/ronykit/std/gateway/tcp"
)

func main() {
	// Create a REST bundle capable of handling REST requests.
	restBundle, err := rest.New(
		tcpGateway.Config{
			Concurrency:   100,
			ListenAddress: "0.0.0.0:80",
		},
	)
	if err != nil {
		panic(err)
	}

	// Implement Echo API
	restBundle.Set(
		fasthttp.MethodGet, "/echo/:randomID",
		func(bag rest.ParamsGetter, data []byte) ronykit.Message {
			m := &echoRequest{}
			if randomID, ok := bag.Get("randomID").(string); ok {
				m.RandomID = utils.StrToInt64(randomID)
			}

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

	// Implement Sum API
	restBundle.Set(
		fasthttp.MethodGet, "/sum/:val1/:val2",
		func(bag rest.ParamsGetter, data []byte) ronykit.Message {
			m := &sumRequest{}
			if val1, ok := bag.Get("val1").(string); ok {
				m.Val1 = utils.StrToInt64(val1)
			}
			if val2, ok := bag.Get("val2").(string); ok {
				m.Val2 = utils.StrToInt64(val2)
			}

			return m
		},
		func(ctx *ronykit.Context) ronykit.Handler {
			req, ok := ctx.Receive().(*sumRequest)
			if !ok {
				ctx.Send(
					&errorMessage{
						Code:    "E01",
						Message: "Request was not echoRequest",
					},
				)

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
	restBundle.Set(
		fasthttp.MethodPost, "/echo",
		func(bag rest.ParamsGetter, data []byte) ronykit.Message {
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
				ctx.Send(
					&errorMessage{
						Code:    "E01",
						Message: "Request was not echoRequest",
					},
				)

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
