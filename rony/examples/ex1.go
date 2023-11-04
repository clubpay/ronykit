package main

import (
	"context"
	"fmt"
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
			rony.RESTName("EchoRequest1"),
		),
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
	ctx *rony.Context[*EchoCounter, string], in EchoRequest,
) (EchoResponse, rony.Error) {
	res := EchoResponse{Message: in.Message}
	ctx.ReduceState("up", func(s *EchoCounter) {
		res.Count = s.Count
	})
	fmt.Println("Echo", in.Message, res.Count)

	return res, nil
}
