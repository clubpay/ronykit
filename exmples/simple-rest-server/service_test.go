package main

import (
	"reflect"
	"testing"

	"github.com/clubpay/ronykit"
)

func TestSampleService(t *testing.T) {
	err := ronykit.NewTestContext().
		SetHandler(echoHandler).
		Input(
			&echoRequest{
				RandomID: 2374,
				Ok:       false,
			},
			nil,
		).
		Expectation(
			func(out ...*ronykit.Envelope) error {
				if len(out) != 1 {
					t.Fatalf("expected 1 envelope, got %d", len(out))
				}
				out1 := out[0]
				res1, ok := out1.GetMsg().(*echoResponse)
				if !ok {
					t.Fatalf("expected echoResponse, got %v", reflect.TypeOf(out1.GetMsg()))
				}
				if res1.RandomID != 2374 {
					t.Fatalf("got unexpected randomID: %d", res1.RandomID)
				}

				return nil
			},
		).Run()
	if err != nil {
		t.Fatal(err)
	}
}
