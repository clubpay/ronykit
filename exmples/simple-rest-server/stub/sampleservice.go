package sampleservicestub

import (
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

type cannedEchoResponse struct {
	EchoResponse *EchoResponse
}

func (s SampleServiceStub) Echo(req *EchoRequest) (cannedEchoResponse, error) {
	res := cannedEchoResponse{
		EchoResponse: &EchoResponse{},
	}
	err := s.s.REST().
		SetMethod("GET").
		DefaultResponseHandler(nil).
		AutoRun("/echo/:randomID", ronykit.JSON, req).
		Err()

	return res, err
}

type cannedSum1Response struct {
	SumResponse *SumResponse
}

func (s SampleServiceStub) Sum1(req *SumRequest) (cannedSum1Response, error) {
	res := cannedSum1Response{
		SumResponse: &SumResponse{},
	}
	err := s.s.REST().
		SetMethod("GET").
		DefaultResponseHandler(nil).
		AutoRun("/sum/:val1/:val2", ronykit.JSON, req).
		Err()

	return res, err
}

type cannedSum2Response struct {
	SumResponse *SumResponse
}

func (s SampleServiceStub) Sum2(req *SumRequest) (cannedSum2Response, error) {
	res := cannedSum2Response{
		SumResponse: &SumResponse{},
	}
	err := s.s.REST().
		SetMethod("POST").
		DefaultResponseHandler(nil).
		AutoRun("/sum", ronykit.JSON, req).
		Err()

	return res, err
}

type cannedSumRedirectResponse struct {
	SumResponse *SumResponse
}

func (s SampleServiceStub) SumRedirect(req *SumRequest) (cannedSumRedirectResponse, error) {
	res := cannedSumRedirectResponse{
		SumResponse: &SumResponse{},
	}
	err := s.s.REST().
		SetMethod("GET").
		DefaultResponseHandler(nil).
		AutoRun("/sum-redirect/:val1/:val2", ronykit.JSON, req).
		Err()

	return res, err
}
