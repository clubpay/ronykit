package stubgen

type Option func(*genConfig)

type genConfig struct {
	genFunc    GenFunc
	pkgName    string
	folderName string
	outputDir  string
	tags       []string
}

var defaultConfig = genConfig{
	genFunc:   GolangStub,
	outputDir: ".", // current directory
}

func WithGenFunc(gf GenFunc) Option {
	return func(c *genConfig) {
		c.genFunc = gf
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
