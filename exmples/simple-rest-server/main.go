package main

import (
	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
	"github.com/valyala/fasthttp"
	"syscall"

	"github.com/ronaksoft/ronykit/std/bundle/rest"
	tcpGateway "github.com/ronaksoft/ronykit/std/gateway/tcp"
)

func main() {
	s := ronykit.NewServer()

	gw := tcpGateway.MustNew(
		tcpGateway.Config{
			ListenAddress: "0.0.0.0:80",
		},
	)

	bundle := rest.New()
	bundle.Set(fasthttp.MethodGet, "/echo/:randomID",
		func(ctx *ronykit.Context) ronykit.Handler {
			req, ok := ctx.Receive().(*echoRequest)
			if !ok {
				_ = ctx.Send(
					&errorMessage{
						Code:    "E01",
						Message: "Request was not echoRequest",
					},
				)
			}

			res := &echoResponse{
				RandomID: req.RandomID,
			}

			_ = ctx.Send(res)

			return nil
		},
	)

	s.RegisterGateway(gw, bundle)

	// Start the server
	s.Start()

	// Wait for signal to shut down
	s.Shutdown(syscall.SIGHUP)
}

type errorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *errorMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *errorMessage) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type echoRequest struct {
	RandomID int64 `json:"randomID"`
}

func (e *echoRequest) Unmarshal(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (e *echoRequest) Marshal() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

type echoResponse struct {
	RandomID int64 `json:"randomID"`
}

func (e *echoResponse) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *echoResponse) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
