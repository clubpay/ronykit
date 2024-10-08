package stubgen

type Option func(*genConfig)

type genConfig struct {
	genFunc GenFunc
	// stubName is name of the stub that will be generated.
	stubName string
	// pkgName is name of the package the is generated.
	// It is useful in languages that need package name like go
	pkgName string
	// folderName is name of the folder that will be created and generated
	// code will be placed in it.
	folderName string
	// outputDir is the directory where the generated code will be placed.
	// final path will be outputDir/folderName
	// default is current directory
	outputDir     string
	fileExtension string
	tags          []string
	// extraOptions are key-values that could be injected to the template for customized
	// outputs.
	extraOptions map[string]string
}

var defaultConfig = genConfig{
	genFunc:       GolangStub,
	fileExtension: ".go",
	outputDir:     ".", // current directory
}

func WithGenFunc(gf GenFunc, ext string) Option {
	return func(c *genConfig) {
		c.genFunc = gf
		c.fileExtension = ext
	}
}

func WithPkgName(name string) Option {
	return func(c *genConfig) {
		c.pkgName = name
	}
}

func WithFolderName(name string) Option {
	return func(c *genConfig) {
		c.folderName = name
	}
}

func WithOutputDir(dir string) Option {
	return func(c *genConfig) {
		c.outputDir = dir
	}
}

func WithTags(tags ...string) Option {
	return func(c *genConfig) {
		c.tags = tags
	}
}

func WithStubName(name string) Option {
	return func(c *genConfig) {
		c.stubName = name
	}
}

func WithExtraOptions(opt map[string]string) Option {
	return func(c *genConfig) {
		c.extraOptions = opt
	}
}
