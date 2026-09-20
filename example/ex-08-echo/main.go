package main

import (
	"context"
	"fmt"
	"os"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/rony"
	"github.com/clubpay/ronykit/x/rkit"
)

func main() {
	srv := rony.NewServer(
		rony.Listen(":81"),
		rony.WithServerName("EchoServer"),
		rony.WithAPIDocs("/docs"),
		rony.WithCORS(rony.CORSConfig{}),
	)

	// Set up the server with the initial state, which is a pointer to EchoCounter
	// We can have as many states as we want. But each handler can only work with
	// one state. In other words, we cannot register one handler with two different
	// setup contexts.
	rony.Setup(
		srv,
		"EchoService",
		rony.EmptyState(),
		// Register the echo handler for both GET /echo and GET /echo/{id}
		// This way all the following requests are valid:
		rony.WithMiddleware[rony.EMPTY](printMW),
		rony.WithUnary(
			echo,
			rony.GET("/echo/{id}"),
			rony.GET("/echo"),
		),
	)

	// Run the server in blocking mode
	err := srv.Run(
		context.Background(),
		os.Kill, os.Interrupt,
	)
	if err != nil {
		panic(err)
	}
}

type EchoRequestDTO struct {
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
}

type EchoResponseDTO struct {
	ID      string `json:"id"`
	Latency int64  `json:"latency"`
}

func echo(_ *rony.UnaryCtx[rony.EMPTY, rony.NOP], req EchoRequestDTO) (*EchoResponseDTO, error) {
	res := &EchoResponseDTO{
		ID:      req.ID,
		Latency: rkit.NanoTime() - req.Timestamp,
	}

	return res, nil
}

func printMW(ctx *kit.Context) {
	fmt.Println("req", rkit.B2S(rkit.Must(kit.MarshalMessage(ctx.In().GetMsg()))))
	ctx.AddModifier(
		func(envelope *kit.Envelope) {
			fmt.Println("res", rkit.B2S(rkit.Must(kit.MarshalMessage(envelope.GetMsg()))))
		},
	)
}
