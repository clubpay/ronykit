package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/clubpay/ronykit/rony"
	"github.com/clubpay/ronykit/rony/errs"
)

func main() {
	srv := rony.NewServer(
		rony.Listen(":80"),
		rony.WithServerName("CounterServer"),
		rony.WithAPIDocs("/docs"),
		rony.UseScalarUI(),
	)

	// Set up the server with the initial state, which is a pointer to EchoCounter
	// We can have as many states as we want. But each handler can only work with
	// one state. In other words, we cannot register one handler with two different
	// setup contexts.
	rony.Setup(
		srv,
		"CounterService",
		rony.ToInitiateState(
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
		rony.WithStream(
			countStream,
			rony.SSE("/sse/count"),
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

type UCtx = rony.UnaryCtx[*EchoCounter, string]

func count(ctx *UCtx, req CounterRequestDTO) (*CounterResponseDTO, error) {
	res := &CounterResponseDTO{}

	fmt.Println(req.Action, req.Count)

	err := ctx.ReduceState(
		req.Action,
		func(s *EchoCounter, err error) error {
			if err != nil {
				return errs.WrapCode(err, errs.InvalidArgument, "BAD_REQUEST")
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

type SCtx = rony.StreamCtx[*EchoCounter, string, CounterResponseDTO]

func countStream(ctx *SCtx, _ CounterRequestDTO) error {
	for {
		ctx.Push(CounterResponseDTO{Count: ctx.State().Count})
		time.Sleep(time.Second)
		ctx.Push(CounterResponseDTO{Count: ctx.State().Count})
		time.Sleep(time.Second)
		break
	}

	return nil
}
