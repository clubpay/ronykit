package main

import (
	"context"
	"fmt"
	"net/http"

	sampleservicestub "github.com/clubpay/ronykit/exmples/simple-rest-server/stub"
	"github.com/clubpay/ronykit/stub"
	"github.com/goccy/go-json"
)

func main() {
	res1 := sampleservicestub.EchoResponse{}
	s := stub.New("127.0.0.1")
	httpCtx := s.REST().
		SetMethod(http.MethodGet).
		SetPath("echo/1230").
		SetQuery("ok", "true").
		SetResponseHandler(
			http.StatusOK,
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {
				return stub.WrapError(json.UnmarshalNoEscape(r.GetBody(), &res1))
			},
		).
		Run(context.Background())
	defer httpCtx.Release()

	if httpCtx.Err() != nil {
		panic(httpCtx.Err())
	}
	//nolint:forbidigo
	fmt.Println("RESPONSE1: ", res1.Ok, res1.RandomID)

	s2 := sampleservicestub.NewSampleServiceStub(
		"127.0.0.1",
	)
	res2, err := s2.Echo(
		context.Background(),
		&sampleservicestub.EchoRequest{
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
