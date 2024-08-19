package main

import (
	"github.com/clubpay/ronykit/async"
)

func main() {
	e, err := async.NewEngine(
		nil,
		async.Register(),
	)
	if err != nil {
		panic(err)
	}
}

type TD1 struct{}

func (td TD1) MarshalBinary() ([]byte, error) { return nil, nil }

func (td *TD1) UnmarshalBinary([]byte) error { return nil }

var T1 = async.SetupTask[TD1](
	"t1",
	func(ctx *async.Context, p TD1) error {

		return nil
	},
)
