package main

import (
	"testing"

	"github.com/clubpay/ronykit/example/ex-01-rpc/dto"
	"github.com/clubpay/ronykit/kit"
	"github.com/stretchr/testify/assert"
)

func TestEcho(t *testing.T) {
	var (
		reqDTO = &dto.EchoRequest{
			RandomID: 2374,
			Ok:       false,
		}
		resDTO *dto.EchoResponse
		errDTO *dto.ErrorMessage
	)
	err := kit.Expect(
		kit.NewTestContext().SetHandler(echoHandler),
		reqDTO, &resDTO, &errDTO,
	)
	if err != nil {
		t.Fatal(err)
	}
	assert.Nil(t, errDTO)
	assert.Equal(t, reqDTO.RandomID, resDTO.RandomID)
	assert.Equal(t, reqDTO.Ok, resDTO.Ok)
}
