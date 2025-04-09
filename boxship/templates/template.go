package templates

import "embed"

//go:embed default/*
var EnvFolder embed.FS
