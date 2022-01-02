package jsonrpc

import "github.com/ronaksoft/ronykit"

type routerData struct {
	ServiceName string
	Handlers    []ronykit.Handler
}

type router struct {
	routes map[string]routerData
}
