//go:build ignore

package main

import (
	"github.com/clubpay/ronykit/example/ex-04-stubgen/api"
	"github.com/clubpay/ronykit/stub/stubgen"
)

func main() {
	stubgen.New(
		stubgen.WithTags("json"),
		stubgen.WithPkgName("sampleservice"),
		stubgen.WithFolderName("sampleservice"),
		stubgen.WithStubName("sampleService"),
	).MustGenerate(api.SampleDesc)

	stubgen.New(
		stubgen.WithGenFunc(stubgen.TypeScriptStub, ".ts"),
		stubgen.WithTags("json"),
		stubgen.WithPkgName("sampleservice"),
		stubgen.WithFolderName("sampleservicets"),
		stubgen.WithStubName("sampleService"),
	).MustGenerate(api.SampleDesc)
}
