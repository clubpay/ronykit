package main

import (
	"context"
	"os"
	"sync"

	"github.com/clubpay/ronykit/rony"
)

func main() {
	srv := rony.NewServer(
		rony.Listen(":8080"),
		rony.WithCORS(
			rony.CORSConfig{
				AllowedOrigins: []string{"*"},
			},
		),
		rony.WithServerName("RonyExample"),
		rony.WithCompression(rony.CompressionLevelBestCompression),
		rony.WithWebsocketEndpoint("/ws"),
	)

	setup := rony.Setup(
		srv, rony.ToInitiateState[*EchoCounter, string](
			&EchoCounter{
				Count: 19,
			},
		),
	)

	rony.RegisterUnary(
		setup, echo,
		rony.GET("/echo",
			rony.UnaryName("EchoWithQueryParam"),
		),
		rony.GET("/echo/{message}",
			rony.UnaryName("EchoWithURLParam"),
		),
	)

	rony.RegisterStream(
		setup, echoN,
		rony.RPC("echo"),
		rony.RPC("echoN"),
	)

	err := srv.SwaggerAPI("_swagger.json")
	if err != nil {
		panic(err)
	}
	err = srv.Run(context.Background(), os.Interrupt, os.Kill)
	if err != nil {
		panic(err)
	}
}

type EchoRequest struct {
	Qty     int    `json:"qty"`
	Message string `json:"message"`
}

type EchoResponse struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type EchoCounter struct {
	sync.Mutex

	Count int
}

func (e *EchoCounter) Name() string {
	return "EchoCounter"
}

func (e *EchoCounter) Reduce(action string) {
	switch action {
	case "up":
		e.Count++
	case "down":
		e.Count--
	}
}

func echo(
	ctx *rony.UnaryCtx[*EchoCounter, string], in EchoRequest,
) (EchoResponse, rony.Error) {
	res := EchoResponse{Message: in.Message}
	ctx.ReduceState("up", func(s *EchoCounter) {
		res.Count = s.Count
	})

	return res, nil
}

func echoN(
	ctx *rony.StreamCtx[*EchoCounter, string, EchoResponse], in EchoRequest,
) error {
	for i := 0; i < in.Qty; i++ {
		res := EchoResponse{Message: in.Message}
		ctx.ReduceState("up", func(s *EchoCounter) {
			res.Count = s.Count
		})

		ctx.Push(
			res,
			rony.WithHdr("Route", ctx.Route()),
		)
	}

	return nil
}
