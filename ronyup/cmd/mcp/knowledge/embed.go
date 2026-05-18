package knowledge

import "embed"

//go:embed all:resources/architecture all:resources/characteristics all:resources/packages all:prompts all:server all:resources/tools
var content embed.FS
