package main

import (
	"context"
	"net/http"

	sampleservicestub "github.com/clubpay/ronykit/exmples/simple-rest-server/stub"
	"github.com/clubpay/ronykit/stub"
	"github.com/goccy/go-json"
)

func main() {
	res := sampleservicestub.EchoResponse{}
	s := stub.New("127.0.0.1")
	err := s.REST().
		SetMethod(http.MethodGet).
		SetPath("echo/1230").
		SetResponseHandler(
			http.StatusOK,
			func(ctx context.Context, r stub.RESTResponse) error {
				return json.Unmarshal(r.GetBody(), &res)
			},
		).
		Run(context.Background()).
		Err()
	if err != nil {
		panic(err)
	}
	if res.RandomID != 1230 {
		panic("random is not correct")
	}

	s2 := sampleservicestub.NewSampleServiceStub("127.0.0.1")
	s2.Echo(
		&sampleservicestub.EchoRequest{
			RandomID: 1450,
			Ok:       false,
		},
	)
}
