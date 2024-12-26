package main

import (
	"context"
	"fmt"
	"os"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/rony"
)

func main() {
	srv := rony.NewServer(
		rony.Listen(":80"),
		rony.WithServerName("EchoServer"),
	)

	// Set up the server with the initial state, which is a pointer to EchoCounter
	// We can have as many states as we want. But each handler can only work with
	// one state. In other words, we cannot register one handler with two different
	// setup contexts.
	rony.Setup(
		srv,
		"EchoService",
		rony.ToInitiateState[*Counter, rony.NOP](&Counter{}),
		// Register the echo handler for both GET /echo and GET /echo/{id}
		// This way all the following requests are valid:
		rony.WithMiddleware[*Counter, rony.NOP](printMW),
		rony.WithMiddleware[
			*Counter, rony.NOP, rony.StatefulMiddleware[*Counter, rony.NOP],
		](printState),
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

type Counter struct {
	Count int64
}

var _ rony.State[rony.NOP] = (*Counter)(nil)

func (c *Counter) Name() string {
	return "Counter"
}

func (c *Counter) Reduce(_ rony.NOP) error {
	c.Count++

	return nil
}

type EchoRequestDTO struct {
	ID        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
}

type EchoResponseDTO struct {
	ID      string `json:"id"`
	Latency int64  `json:"latency"`
}

func echo(ctx *rony.UnaryCtx[*Counter, rony.NOP], req EchoRequestDTO) (*EchoResponseDTO, error) {
	res := &EchoResponseDTO{
		ID:      req.ID,
		Latency: utils.NanoTime() - req.Timestamp,
	}

	_ = ctx.ReduceState(rony.NOP{}, nil)

	kit.ContextWithValue(ctx.Context(), "MyKey", "Hi")

	return res, nil
}

// printMW is a stateless middleware that prints the request and response
func printMW(ctx *kit.Context) {
	fmt.Println("req", utils.B2S(utils.Must(kit.MarshalMessage(ctx.In().GetMsg()))))
	ctx.AddModifier(
		func(envelope *kit.Envelope) {
			fmt.Println("res", utils.B2S(utils.Must(kit.MarshalMessage(envelope.GetMsg()))))
		},
	)
	ctx.Next()

	fmt.Println(ctx.Get("MyKey"))
}

// printState is a stateful middleware that prints the state of the counter
func printState(ctx *rony.BaseCtx[*Counter, rony.NOP]) {
	fmt.Println("count:", ctx.State().Count)
}
