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

	setup := rony.Setup[string](
		srv,
		func() *EchoCounter {
			return &EchoCounter{
				Count: 19,
			}
		},
	)

	rony.RegisterHandler(
		setup,
		rony.GET("/echo"),
		echo,
	)

	err := srv.Run(context.Background(), os.Interrupt, os.Kill)
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
	mu    sync.Mutex
	Count int
}

func (e *EchoCounter) Name() string {
	return "EchoCounter"
}

func (e *EchoCounter) Reduce(action string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	switch action {
	case "up":
		e.Count++
	case "down":
		e.Count--
	}
}

func echo(
	ctx *rony.Context[string, *EchoCounter], in EchoRequest,
) (EchoResponse, rony.Error) {
	s := ctx.State()
	s.Reduce("up")

	fmt.Println("Echo", in.Message, s.Count)

	return EchoResponse{Message: in.Message, Count: s.Count}, nil
}
