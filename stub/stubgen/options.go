package stubgen

type Option func(*genConfig)

type genConfig struct {
	genEngine GenEngine
	// stubName is the name of the stub that will be generated.
	stubName string
	// folderName is the name of the folder that will be created, and generated
	// code will be placed in it.
	folderName string
	// outputDir is the directory where the generated code will be placed.
	// the final path will be outputDir/folderName
	// default is the current directory
	outputDir string
	tags      []string
}

var defaultConfig = genConfig{
	outputDir: ".", // current directory
}

func WithGenEngine(gf GenEngine) Option {
	return func(c *genConfig) {
		c.genEngine = gf
	}
}

// WithFolderName sets the folder name where the generated code will be placed.
func WithFolderName(name string) Option {
	return func(c *genConfig) {
		c.folderName = name
	}
}

// WithOutputDir sets the directory where the generated code will be placed.
// The final path will be outputDir/folderName.
//
//	Default is the current directory.
func WithOutputDir(dir string) Option {
	return func(c *genConfig) {
		c.outputDir = dir
	}
}

// WithTags sets the tags for the generated code option.
func WithTags(tags ...string) Option {
	return func(c *genConfig) {
		c.tags = tags
	}
}

// WithStubName sets the name of the stub that will be generated.
func WithStubName(name string) Option {
	return func(c *genConfig) {
		c.stubName = name
	}
}
