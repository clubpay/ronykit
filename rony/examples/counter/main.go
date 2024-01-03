package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/clubpay/ronykit/rony"
)

func main() {
	srv := rony.NewServer(
		rony.Listen(":80"),
		rony.WithServerName("CounterServer"),
	)

	// Set up the server with the initial state, which is a pointer to EchoCounter
	// We can have as many states as we want. But each handler can only work with
	// one state. In other words, we cannot register one handler with two different
	// setup contexts.
	rony.Setup(
		srv,
		"CounterService",
		rony.ToInitiateState[*EchoCounter, string](
			&EchoCounter{
				Count: 0,
			},
		),
		// Register the count handler for both GET /count and GET /count/{action}
		// This way all the following requests are valid:
		// 1. GET /count/up&count=1
		// 2. GET /count/down&count=2
		// 3. GET /count?action=up&count=1
		// 4. GET /count?action=down&count=2
		rony.WithUnary(
			count,
			rony.GET("/count/{action}"),
			rony.GET("/count"),
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

type EchoCounter struct {
	sync.Mutex

	Count int
}

func (e *EchoCounter) Name() string {
	return "EchoCounter"
}

func (e *EchoCounter) Reduce(action string) error {
	switch strings.ToLower(action) {
	case "up":
		e.Count++
	case "down":
		if e.Count <= 0 {
			return fmt.Errorf("count cannot be negative")
		}
		e.Count--
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	return nil
}

type CounterRequestDTO struct {
	Action string `json:"action"`
	Count  int    `json:"count"`
}

type CounterResponseDTO struct {
	Count int `json:"count"`
}

func count(ctx *rony.UnaryCtx[*EchoCounter, string], req CounterRequestDTO) (*CounterResponseDTO, error) {
	res := &CounterResponseDTO{}
	fmt.Println(req.Action, req.Count)
	err := ctx.ReduceState(
		req.Action,
		func(s *EchoCounter, err error) error {
			if err != nil {
				return rony.NewError(err).SetCode(http.StatusBadRequest)
			}

			res.Count = s.Count

			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return res, nil
}
