package edge

import "github.com/ronaksoft/ronykit"

type Router interface {
	Route(envelope ronykit.Envelope) ([]Handler, error)
}
