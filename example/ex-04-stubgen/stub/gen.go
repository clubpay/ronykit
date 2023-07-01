//go:build ignore

package main

import (
	"github.com/clubpay/ronykit/example/ex-04-stubgen/api"
	"github.com/clubpay/ronykit/kit/stub/stubgen"
)

func main() {
	stubgen.New(
		stubgen.WithTags("json"),
		stubgen.WithPkgName("sampleservice"),
		stubgen.WithFolderName("sampleservice"),
		stubgen.WithStubName("sampleService"),
	).MustGenerate(api.SampleDesc)
}
