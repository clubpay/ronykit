package rest

import "github.com/ronaksoft/ronykit/std/bundle/rest/mux"

type routeData struct {
	method  string
	path    string
	decoder mux.DecoderFunc
}

func NewRouteData(method string, path string, decoder mux.DecoderFunc) *routeData {
	return &routeData{
		method:  method,
		path:    path,
		decoder: decoder,
	}
}

func (r routeData) Query(q string) interface{} {
	switch q {
	case queryDecoder:
		return r.decoder
	case queryMethod:
		return r.method
	case queryPath:
		return r.path
	}

	return nil
}
