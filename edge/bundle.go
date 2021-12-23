package edge

import "github.com/ronaksoft/ronykit"

type Bundle struct {
	Dispatcher   ronykit.Dispatcher
	EnvelopePool ronykit.EnvelopePool
	Router       Router
}
