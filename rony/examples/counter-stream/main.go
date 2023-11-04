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
		rony.WithWebsocketEndpoint("/ws"),
		rony.WithPredicateKey("cmd"),
	)

	// Set up the server with the initial state, which is a pointer to EchoCounter
	// We can have as many states as we want. But each handler can only work with
	// one state. In other words, we cannot register one handler with two different
	// setup contexts.
	setupCtx := rony.Setup(
		srv,
		rony.ToInitiateState[*EchoCounter, string](
			&EchoCounter{
				Count: 0,
			},
		),
	)

	// Register the count handler for both GET /count and GET /count/{action}
	// This way all the following requests are valid:
	// 1. GET /count/up&count=1
	// 2. GET /count/down&count=2
	// 3. GET /count?action=up&count=1
	// 4. GET /count?action=down&count=2
	rony.RegisterUnary(
		setupCtx, count,
		rony.GET("/count/{action}"),
		rony.GET("/count"),
	)

	// Register the count handler for Websocket messages
	// This way all the following requests are valid:
	// Websocket /ws
	// {
	//   "hdr": {
	//     "cmd": "count",
	//   },
	//   "payload": {
	//     "action": "up",
	//     "count": 1
	//   }
	// }
	rony.RegisterStream(
		setupCtx, countStream,
		rony.RPC("count"),
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

func countStream(ctx *rony.StreamCtx[*EchoCounter, string, *CounterResponseDTO], req CounterRequestDTO) error {
	for i := 0; i < req.Count; i++ {
		res := &CounterResponseDTO{}
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
			return err
		}

		ctx.Push(
			res,
			rony.WithHdr("type", "counter-response"),
		)
	}

	return nil
}
