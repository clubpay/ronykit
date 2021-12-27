package jsonrpc

import "github.com/ronaksoft/ronykit"

type routerData struct {
	RequestFactory func() ronykit.Message
	Handlers       []ronykit.Handler
}

type router struct {
	routes map[string]routerData
}
