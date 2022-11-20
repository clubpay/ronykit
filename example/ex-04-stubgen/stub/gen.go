//go:build ignore

package main

import (
	"github.com/clubpay/ronykit/example/ex-04-stubgen/api"
	"github.com/clubpay/ronykit/kit/stub/stubgen"
)

func main() {
	stubgen.New("sampleservice", "json").
		SetFunc(stubgen.GolangStub).
		MustGenerate(api.SampleDesc)
}
