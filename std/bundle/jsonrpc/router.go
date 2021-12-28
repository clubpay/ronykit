package jsonrpc

import "github.com/ronaksoft/ronykit"

type routerData struct {
	Handlers []ronykit.Handler
}

type router struct {
	routes map[string]routerData
}
