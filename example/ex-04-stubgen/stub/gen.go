//go:build ignore

package main

import (
	"github.com/clubpay/ronykit/example/ex-04-stubgen/api"
	"github.com/clubpay/ronykit/stub/stubgen"
)

func main() {
	stubgen.New(
		stubgen.WithGenEngine(stubgen.NewGolangEngine(stubgen.GolangConfig{
			PkgName: "sampleservice",
		})),
		stubgen.WithTags("json"),
		stubgen.WithFolderName("sampleservice"),
		stubgen.WithStubName("sampleService"),
	).MustGenerate(api.SampleDesc)

	stubgen.New(
		stubgen.WithGenEngine(stubgen.NewTypescriptEngine(stubgen.TypescriptConfig{
			GenerateSWR: true,
		})),
		stubgen.WithTags("json"),
		stubgen.WithFolderName("sampleservicets"),
		stubgen.WithStubName("sampleService"),
	).MustGenerate(api.SampleDesc)
}
