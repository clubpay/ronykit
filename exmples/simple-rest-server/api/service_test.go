package api_test

import (
	"reflect"
	"testing"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/exmples/simple-rest-server/api"
	"github.com/clubpay/ronykit/exmples/simple-rest-server/dto"
)

func TestSampleService(t *testing.T) {
	err := ronykit.NewTestContext().
		SetHandler(api.EchoHandler).
		Input(
			&dto.EchoRequest{
				RandomID: 2374,
				Ok:       false,
			},
			nil,
		).
		Receiver(
			func(out ...ronykit.Envelope) error {
				if len(out) != 1 {
					t.Fatalf("expected 1 envelope, got %d", len(out))
				}
				out1 := out[0]
				res1, ok := out1.GetMsg().(*dto.EchoResponse)
				if !ok {
					t.Fatalf("expected echoResponse, got %v", reflect.TypeOf(out1.GetMsg()))
				}
				if res1.RandomID != 2374 {
					t.Fatalf("got unexpected randomID: %d", res1.RandomID)
				}

				return nil
			},
		).Run(false)
	if err != nil {
		t.Fatal(err)
	}
}
