package sampleservicestub

import (
	"context"
	"net/http"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/stub"
)

// EchoRequest is a data transfer object
type EchoRequest struct {
	RandomID int64 `json:"randomID"`
	Ok       bool  `json:"ok"`
}

// EchoResponse is a data transfer object
type EchoResponse struct {
	RandomID int64 `json:"randomID"`
	Ok       bool  `json:"ok"`
}

// EmbeddedHeader is a data transfer object
type EmbeddedHeader struct {
	SomeKey1 string `json:"someKey1"`
	SomeInt1 int64  `json:"someInt1"`
}

// RedirectRequest is a data transfer object
type RedirectRequest struct {
	URL string `json:"url"`
}

// SumRequest is a data transfer object
type SumRequest struct {
	EmbeddedHeader
	Val1 int64 `json:"val1"`
	Val2 int64 `json:"val2"`
}

// SumResponse is a data transfer object
type SumResponse struct {
	EmbeddedHeader
	Val int64
}

// SampleServiceStub represents the client/stub for SampleService.
type SampleServiceStub struct {
	hostPort  string
	secure    bool
	verifyTLS bool

	s *stub.Stub
}

func NewSampleServiceStub(hostPort string, opts ...stub.Option) *SampleServiceStub {
	s := &SampleServiceStub{
		s: stub.New(hostPort, opts...),
	}

	return s
}

func (s SampleServiceStub) Echo(req *EchoRequest) {
	s.s.REST().
		SetMethod("GET").
		DefaultResponseHandler(nil).
		AutoRun("/echo/:randomID", req)
}

func (s SampleServiceStub) Sum1(req *SumRequest) (*SumResponse, error) {
	res := &SumResponse{}
	err := s.s.REST().
		SetMethod("GET").
		SetResponseHandler(
			http.StatusOK,
			func(ctx context.Context, r stub.RESTResponse) error {
				return ronykit.UnmarshalMessage(r.GetBody(), res)
			},
		).
		DefaultResponseHandler(
			func(ctx context.Context, r stub.RESTResponse) error {
				return nil
			},
		).
		AutoRun("/sum/:val1/:val2", req).
		Err()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s SampleServiceStub) Sum2(req *SumRequest) {
	s.s.REST().
		SetMethod("POST").
		SetResponseHandler(
			http.StatusOK,
			func(ctx context.Context, r stub.RESTResponse) error {
				return nil
			},
		).
		DefaultResponseHandler(nil).
		AutoRun("/sum", req)
}

func (s SampleServiceStub) SumRedirect(req *SumRequest) {
	s.s.REST().
		SetMethod("GET").
		DefaultResponseHandler(nil).
		AutoRun("/sum-redirect/:val1/:val2", req)
}
