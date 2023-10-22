package main

import (
	"context"
	"os"
	"sync"

	"github.com/clubpay/ronykit/rony"
)

func main() {
	srv := rony.NewServer()

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
	sync.Mutex
	Count int
}

func (e *EchoCounter) Name() string {
	return "EchoCounter"
}

func (e *EchoCounter) Reduce(action string) {
	e.Lock()
	defer e.Unlock()

	switch action {
	case "up":
		e.Count++
	case "down":
		e.Count--
	}
}

var _ rony.State[string] = (*EchoCounter)(nil)

func echo(
	ctx *rony.Context[string, *EchoCounter, EchoCounter], in EchoRequest,
) (EchoResponse, rony.Error) {
	s := ctx.State()
	s.Reduce("up")

	return EchoResponse{Message: in.Message, Count: s.Count}, nil
}
