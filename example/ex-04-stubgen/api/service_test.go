package api_test

import (
	"reflect"
	"testing"

	"github.com/clubpay/ronykit/example/ex-04-stubgen/api"
	"github.com/clubpay/ronykit/example/ex-04-stubgen/dto"
	"github.com/clubpay/ronykit/kit"
)

func TestSampleService(t *testing.T) {
	err := kit.NewTestContext().
		SetHandler(api.DummyHandler).
		Input(
			&dto.VeryComplexRequest{
				Key1:      "somekey",
				Key1Ptr:   nil,
				MapKey1:   nil,
				MapKey2:   nil,
				SliceKey1: nil,
				SliceKey2: nil,
			},
			nil,
		).
		Receiver(
			func(out ...*kit.Envelope) error {
				if len(out) != 1 {
					t.Fatalf("expected 1 envelope, got %d", len(out))
				}
				out1 := out[0]
				res1, ok := out1.GetMsg().(*dto.VeryComplexResponse)
				if !ok {
					t.Fatalf("expected echoResponse, got %v", reflect.TypeOf(out1.GetMsg()))
				}
				if res1.Key1 != "somekey" {
					t.Fatalf("got unexpected randomID: %s", res1.Key1)
				}

				return nil
			},
		).Run(false)
	if err != nil {
		t.Fatal(err)
	}
}
