package mcp

import (
	"fmt"

	"github.com/clubpay/ronykit/kit"
)

const (
	queryName        = "name"
	queryTitle       = "title"
	queryDesc        = "desc"
	queryDestructive = "destructive"
	queryOpenWorld   = "openWorld"
	queryIdempotent  = "idempotent"
	queryReadOnly    = "readonly"
)

type Selector struct {
	Name        string
	Title       string
	Description string

	Destructive *bool
	OpenWorld   *bool
	Idempotent  bool
	ReadOnly    bool
}

var (
	_ kit.RouteSelector    = (*Selector)(nil)
	_ kit.RPCRouteSelector = (*Selector)(nil)
)

func (s Selector) GetPredicate() string {
	return s.Name
}

func (s Selector) Query(q string) any {
	switch q {
	case queryName:
		return s.Name
	case queryTitle:
		return s.Title
	case queryDesc:
		return s.Description
	case queryDestructive:
		return s.Destructive
	case queryOpenWorld:
		return s.OpenWorld
	case queryIdempotent:
		return s.Idempotent
	case queryReadOnly:
		return s.ReadOnly
	}

	panic(fmt.Errorf("unknown query: %s", q))
}

func (s Selector) GetEncoding() kit.Encoding {
	return kit.JSON
}

func (s Selector) String() string {
	return fmt.Sprintf("%s - %s", s.Name, s.Title)
}
