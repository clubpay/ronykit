package knowledge

import "embed"

//go:embed all:server all:tools all:packages all:architecture all:characteristics all:planning all:prompts
var content embed.FS
