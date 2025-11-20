package services

import (
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
	"github.com/clubpay/ronykit/std/gateways/fastws"
)

type EchoRequest struct {
	Embedded
	Input string `json:"input"`
}

type EchoResponse struct {
	Embedded
	Output      string     `json:"output"`
	MyInterface XInterface `json:"interface"`
}

type XInterface interface {
	X() string
}

type Embedded struct {
	X    string   `json:"x"`
	XP   string   `json:"xp"`
	Y    int64    `json:"y"`
	Z    float64  `json:"z"`
	A    []byte   `json:"a"`
	XPtr *string  `json:"xPtr"`
	YPtr *int64   `json:"yPtr"`
	ZPtr *float64 `json:"zPtr"`
	APtr *[]byte  `json:"aPtr"`
}

type CloseRequest struct {
	Close bool `json:"close"`
}

type CloseResponse struct {
	Close bool `json:"close"`
}

var EchoService kit.ServiceBuilder = desc.NewService("EchoService").
	AddContract(
		desc.NewContract().
			SetInput(&EchoRequest{}).
			SetOutput(&EchoResponse{}).
			AddRoute(desc.Route("", fastws.RPC("echo"))).
			AddRoute(desc.Route("", fasthttp.RPC("echo"))).
			AddRoute(desc.Route("", fasthttp.GET("/echo/{xp}"))).
			SetHandler(
				contextMW(10*time.Second),
				func(ctx *kit.Context) {
					req, _ := ctx.In().GetMsg().(*EchoRequest)

					ctx.In().Reply().
						SetMsg(
							&EchoResponse{
								Embedded: req.Embedded,
								Output:   req.Input,
							},
						).
						Send()
				}),
	).
	AddContract(
		desc.NewContract().
			SetInput(&CloseRequest{}).
			SetOutput(&CloseResponse{}).
			AddRoute(desc.Route("", fastws.RPC("close"))).
			AddRoute(desc.Route("", fasthttp.RPC("close"))).
			SetHandler(
				contextMW(10*time.Second),
				func(ctx *kit.Context) {
					conn, ok := ctx.Conn().(kit.RPCConn)
					if ok {
						conn.Close()
					}

					ctx.Out().
						SetMsg(
							&CloseResponse{
								Close: ok,
							},
						).
						Send()
				}))
