package rony

import (
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/stub/stubgen"
)

// GenerateStub generates a stub file for the given service description.
// The generated file will be placed in the `outputDir/folderName` directory.
// The package name of the generated file will be `pkgName`.
// The struct holding the stub functions will be named `name`.
func GenerateStub[S State[A], A Action](
	name, folderName, outputDir string,
	genEngine stubgen.GenEngine,
	opt ...SetupOption[S, A],
) error {
	var s S

	ctx := SetupContext[S, A]{
		s:    utils.ValPtr(ToInitiateState(s)()),
		name: name,
		cfg:  utils.ValPtr(defaultServerConfig()),
	}
	for _, o := range opt {
		o(&ctx)
	}

	return stubgen.New(
		stubgen.WithGenEngine(genEngine),
		stubgen.WithStubName(utils.ToCamel(name)),
		stubgen.WithFolderName(folderName),
		stubgen.WithOutputDir(outputDir),
	).Generate(ctx.cfg.allServiceDesc()...)
}
