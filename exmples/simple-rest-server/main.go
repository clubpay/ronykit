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
	restServer, err := rest.New(
		tcpGateway.Config{
			Concurrency:   100,
			ListenAddress: "0.0.0.0:80",
		},
	)
	if err != nil {
		panic(err)
	}

	restServer.Set(
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

	ronykit.NewServer(
		ronykit.WithLogger(log.DefaultLogger),
		ronykit.RegisterBundle(restServer),
	).
		Start().
		Shutdown(syscall.SIGHUP)
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
