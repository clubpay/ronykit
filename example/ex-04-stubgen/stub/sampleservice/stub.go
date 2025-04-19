// Code generated by RonyKIT Stub Generator (Golang); DO NOT EDIT.

package sampleservice

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/reflector"
	"github.com/clubpay/ronykit/stub"
)

var (
	_ fmt.Stringer
	_ utils.Result
	_ json.RawMessage
)

func init() {
	reflector.Register(&ErrorMessage{}, "json")
	reflector.Register(&KeyValue{}, "json")
	reflector.Register(&SimpleHdr{}, "json")
	reflector.Register(&VeryComplexRequest{}, "json")
	reflector.Register(&VeryComplexResponse{}, "json")
}

// ErrorMessage is a data transfer object
type ErrorMessage struct {
	Code int    `json:"code"`
	Item string `json:"item"`
}

func (x ErrorMessage) GetCode() int {
	return x.Code
}

func (x ErrorMessage) GetItem() string {
	return x.Item
}

// KeyValue is a data transfer object
type KeyValue struct {
	Key   string `json:"key"`
	Value int    `json:"value"`
}

// SimpleHdr is a data transfer object
type SimpleHdr struct {
	Key1 string      `json:"sKey1"`
	Key2 int         `json:"sKey2"`
	T1   time.Time   `json:"t1"`
	T2   *time.Time  `json:"t2"`
	T3   []time.Time `json:"t3"`
}

// VeryComplexRequest is a data transfer object
type VeryComplexRequest struct {
	SimpleHdr
	Key1      string             `json:"key1"`
	Key1Ptr   *string            `json:"key1Ptr"`
	Key2Ptr   *int               `json:"key2Ptr,omitempty"`
	MapKey1   map[string]int     `json:"mapKey1"`
	MapKey2   map[int64]KeyValue `json:"mapKey2"`
	SliceKey1 []bool             `json:"sliceKey1"`
	SliceKey2 []*KeyValue        `json:"sliceKey2"`
	RawKey    kit.JSONMessage    `json:"rawKey"`
}

// VeryComplexResponse is a data transfer object
type VeryComplexResponse struct {
	Key1      string              `json:"key1,omitempty"`
	Key1Ptr   *string             `json:"key1Ptr,omitempty"`
	MapKey1   map[string]int      `json:"mapKey1,omitempty"`
	MapKey2   map[int64]*KeyValue `json:"mapKey2,omitempty"`
	SliceKey1 []uint8             `json:"sliceKey1"`
	SliceKey2 []KeyValue          `json:"sliceKey2"`
}

type IsampleServiceStub interface {
	ComplexDummy(
		ctx context.Context, req *VeryComplexRequest, opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error)
	ComplexDummy2(
		ctx context.Context, req *VeryComplexRequest, opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error)
	GetComplexDummy(
		ctx context.Context, req *VeryComplexRequest, opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error)
}

// sampleServiceStub represents the client/stub for sampleService.
// Implements IsampleServiceStub
type sampleServiceStub struct {
	hostPort  string
	secure    bool
	verifyTLS bool

	s *stub.Stub
}

func NewsampleServiceStub(hostPort string, opts ...stub.Option) *sampleServiceStub {
	s := &sampleServiceStub{
		s: stub.New(hostPort, opts...),
	}

	return s
}

var _ IsampleServiceStub = (*sampleServiceStub)(nil)

func (s sampleServiceStub) ComplexDummy(
	ctx context.Context, req *VeryComplexRequest, opt ...stub.RESTOption,
) (*VeryComplexResponse, *stub.Error) {

	res := &VeryComplexResponse{}

	httpCtx := s.s.REST(opt...).
		SetMethod("POST").
		SetResponseHandler(
			400,
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {
				res := &ErrorMessage{}
				err := stub.WrapError(kit.UnmarshalMessage(r.GetBody(), res))
				if err != nil {
					return err
				}

				return stub.NewErrorWithMsg(res)
			},
		).
		SetOKHandler(
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {

				return stub.WrapError(kit.UnmarshalMessage(r.GetBody(), res))

			},
		).
		DefaultResponseHandler(
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {
				return stub.NewError(r.StatusCode(), string(r.GetBody()))
			},
		).
		AutoRun(ctx, "/complexDummy", kit.CustomEncoding("json"), req)
	defer httpCtx.Release()

	if err := httpCtx.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

func (s sampleServiceStub) ComplexDummy2(
	ctx context.Context, req *VeryComplexRequest, opt ...stub.RESTOption,
) (*VeryComplexResponse, *stub.Error) {

	res := &VeryComplexResponse{}

	httpCtx := s.s.REST(opt...).
		SetMethod("POST").
		SetResponseHandler(
			400,
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {
				res := &ErrorMessage{}
				err := stub.WrapError(kit.UnmarshalMessage(r.GetBody(), res))
				if err != nil {
					return err
				}

				return stub.NewErrorWithMsg(res)
			},
		).
		SetOKHandler(
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {

				return stub.WrapError(kit.UnmarshalMessage(r.GetBody(), res))

			},
		).
		DefaultResponseHandler(
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {
				return stub.NewError(r.StatusCode(), string(r.GetBody()))
			},
		).
		AutoRun(ctx, "/complexDummy/{key1}", kit.CustomEncoding("json"), req)
	defer httpCtx.Release()

	if err := httpCtx.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

func (s sampleServiceStub) GetComplexDummy(
	ctx context.Context, req *VeryComplexRequest, opt ...stub.RESTOption,
) (*VeryComplexResponse, *stub.Error) {

	res := &VeryComplexResponse{}

	httpCtx := s.s.REST(opt...).
		SetMethod("GET").
		SetResponseHandler(
			400,
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {
				res := &ErrorMessage{}
				err := stub.WrapError(kit.UnmarshalMessage(r.GetBody(), res))
				if err != nil {
					return err
				}

				return stub.NewErrorWithMsg(res)
			},
		).
		SetOKHandler(
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {

				return stub.WrapError(kit.UnmarshalMessage(r.GetBody(), res))

			},
		).
		DefaultResponseHandler(
			func(ctx context.Context, r stub.RESTResponse) *stub.Error {
				return stub.NewError(r.StatusCode(), string(r.GetBody()))
			},
		).
		AutoRun(ctx, "/complexDummy/{key1}/xs/{sKey1}", kit.CustomEncoding("json"), req)
	defer httpCtx.Release()

	if err := httpCtx.Err(); err != nil {
		return nil, err
	}

	return res, nil
}

type MockOption func(*sampleServiceStubMock)

func MockComplexDummy(
	f func(
		ctx context.Context,
		req *VeryComplexRequest,
		opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error),
) MockOption {
	return func(sm *sampleServiceStubMock) {
		sm.complexdummy = f
	}
}

func MockComplexDummy2(
	f func(
		ctx context.Context,
		req *VeryComplexRequest,
		opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error),
) MockOption {
	return func(sm *sampleServiceStubMock) {
		sm.complexdummy2 = f
	}
}

func MockGetComplexDummy(
	f func(
		ctx context.Context,
		req *VeryComplexRequest,
		opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error),
) MockOption {
	return func(sm *sampleServiceStubMock) {
		sm.getcomplexdummy = f
	}
}

// sampleServiceStubMock represents the mocked for client/stub for sampleService.
// Implements IsampleServiceStub
type sampleServiceStubMock struct {
	complexdummy func(
		ctx context.Context,
		req *VeryComplexRequest,
		opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error)
	complexdummy2 func(
		ctx context.Context,
		req *VeryComplexRequest,
		opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error)
	getcomplexdummy func(
		ctx context.Context,
		req *VeryComplexRequest,
		opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error)
}

func NewsampleServiceStubMock(opts ...MockOption) *sampleServiceStubMock {
	s := &sampleServiceStubMock{}
	for _, o := range opts {
		o(s)
	}

	return s
}

var _ IsampleServiceStub = (*sampleServiceStubMock)(nil)

func (s *sampleServiceStubMock) ComplexDummy(
	ctx context.Context,
	req *VeryComplexRequest,
	opt ...stub.RESTOption,
) (*VeryComplexResponse, *stub.Error) {
	if s.complexdummy == nil {
		return nil, stub.WrapError(fmt.Errorf("method not mocked"))
	}

	return s.complexdummy(ctx, req, opt...)
}

func (s *sampleServiceStubMock) SetComplexDummy(
	f func(
		ctx context.Context,
		req *VeryComplexRequest,
		opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error),
) *sampleServiceStubMock {
	s.complexdummy = f

	return s
}

func (s *sampleServiceStubMock) ComplexDummy2(
	ctx context.Context,
	req *VeryComplexRequest,
	opt ...stub.RESTOption,
) (*VeryComplexResponse, *stub.Error) {
	if s.complexdummy2 == nil {
		return nil, stub.WrapError(fmt.Errorf("method not mocked"))
	}

	return s.complexdummy2(ctx, req, opt...)
}

func (s *sampleServiceStubMock) SetComplexDummy2(
	f func(
		ctx context.Context,
		req *VeryComplexRequest,
		opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error),
) *sampleServiceStubMock {
	s.complexdummy2 = f

	return s
}

func (s *sampleServiceStubMock) GetComplexDummy(
	ctx context.Context,
	req *VeryComplexRequest,
	opt ...stub.RESTOption,
) (*VeryComplexResponse, *stub.Error) {
	if s.getcomplexdummy == nil {
		return nil, stub.WrapError(fmt.Errorf("method not mocked"))
	}

	return s.getcomplexdummy(ctx, req, opt...)
}

func (s *sampleServiceStubMock) SetGetComplexDummy(
	f func(
		ctx context.Context,
		req *VeryComplexRequest,
		opt ...stub.RESTOption,
	) (*VeryComplexResponse, *stub.Error),
) *sampleServiceStubMock {
	s.getcomplexdummy = f

	return s
}
