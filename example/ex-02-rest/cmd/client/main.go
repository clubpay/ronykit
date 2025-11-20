package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/clubpay/ronykit/example/ex-02-rest/dto"
	"github.com/clubpay/ronykit/example/ex-02-rest/stub/sampleservice"
	"github.com/clubpay/ronykit/stub"
)

func main() {
	res1 := dto.EchoResponse{}
	s := stub.New("127.0.0.1")

	httpCtx := s.REST().
		SetMethod(http.MethodGet).
		SetPath("echo/1230").
		SetQuery("ok", "true").
		SetResponseHandler(
			http.StatusOK,
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {
				return stub.WrapError(json.Unmarshal(r.GetBody(), &res1))
			},
		).
		Run(context.Background())
	defer httpCtx.Release()

	if httpCtx.Err() != nil {
		panic(httpCtx.Err())
	}
	//nolint:forbidigo
	fmt.Println("RESPONSE1: ", res1.Ok, res1.RandomID)

	/*
		Using the auto-generated service stub
	*/
	s2 := sampleservice.NewStub(
		"127.0.0.1",
	)

	res2, err := s2.EchoGET(
		context.Background(),
		&sampleservice.EchoRequest{
			RandomID: 1450,
			Ok:       true,
		},
	)
	if err != nil {
		panic(err)
	}

	//nolint:forbidigo
	fmt.Println("RESPONSE2: ", res2.Ok, res2.RandomID)
}
