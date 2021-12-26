package rest

import (
	"fmt"

	"github.com/ronaksoft/ronykit"
	tcpGateway "github.com/ronaksoft/ronykit/std/gateway/tcp"
)

var (
	ErrRouteNotFound = fmt.Errorf("route not found")
)

type (
	// ParamsSetter is the interface which should be implemented by the
	// params writer for `search` in order to store the found named path parameters, if any.
	ParamsSetter interface {
		Set(string, interface{})
	}

	// ParamsGetter is the interface which is used on the DecoderFunc to get the extracted data
	// from path parameters.
	ParamsGetter interface {
		Get(key string) interface{}
	}
	// DecoderFunc is the function which gets the raw http's body and the extracted data from
	// path and must return a ronykit.Message.
	DecoderFunc func(bag ParamsGetter, data []byte) ronykit.Message
)

type rest struct {
	gw     *tcpGateway.Gateway
	routes map[string]*trie
}

func New(gatewayConfig tcpGateway.Config, opts ...Option) (*rest, error) {
	r := &rest{
		routes: map[string]*trie{},
	}

	var err error
	r.gw, err = tcpGateway.New(gatewayConfig)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(r)
	}

	return r, nil
}

func (r rest) Dispatch(conn ronykit.Conn, streamID int64, in []byte) ronykit.DispatchFunc {
	rc, ok := conn.(ronykit.REST)
	if !ok {
		return nil
	}

	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		nodeData, err := r.route(rc.GetMethod(), rc.GetPath(), conn)
		if err != nil {
			return err
		}

		writeFunc := func(m ronykit.Message) error {
			data, err := m.Marshal()
			if err != nil {
				return err
			}

			ctx.Walk(
				func(key string, val interface{}) bool {
					if v, ok := val.(string); ok {
						rc.WriteHeader(key, v)
					}
					
					return true
				},
			)

			return conn.Write(streamID, data)
		}

		execFunc(nodeData.decoder(conn, in), writeFunc, nodeData.handlers...)

		return nil
	}

}

func (r rest) Gateway() ronykit.Gateway {
	return r.gw
}

func (r *rest) Dispatcher() ronykit.Dispatcher {
	return r
}
