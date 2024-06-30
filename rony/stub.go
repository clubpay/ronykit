package rony

import "github.com/clubpay/ronykit/stub/stubgen"

// GenerateStub generates a stub file for the given service description.
// The generated file will be placed in the `outputDir/folderName` directory.
// The package name of the generated file will be `pkgName`.
// The struct holding the stub functions will be named `name`.
func GenerateStub[S State[A], A Action](
	name, folderName, outputDir, pkgName string,
	genFunc stubgen.GenFunc, ext string,
	opt ...SetupOption[S, A],
) error {
	ctx := SetupContext[S, A]{
		cfg: &serverConfig{},
	}
	for _, o := range opt {
		o(&ctx)
	}

	return stubgen.New(
		stubgen.WithStubName(name),
		stubgen.WithFolderName(folderName),
		stubgen.WithOutputDir(outputDir),
		stubgen.WithPkgName(pkgName),
		stubgen.WithGenFunc(genFunc, ext),
	).Generate(ctx.cfg.allServiceDesc()...)
}
