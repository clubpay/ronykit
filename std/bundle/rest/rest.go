package rest

import (
	"fmt"

	"github.com/ronaksoft/ronykit"
	tcpGateway "github.com/ronaksoft/ronykit/std/gateway/tcp"
)

var (
	ErrConnectionIsNoREST = fmt.Errorf("connection is not REST")
	ErrRouteNotFound      = fmt.Errorf("route not found")
)

type (
	ParamsGetter interface {
		Get(key string) interface{}
	}
	DecoderFunc func(bag ParamsGetter, data []byte) ronykit.Message
)

type rest struct {
	gw      *tcpGateway.Gateway
	routes  map[string]*trie
	decoder DecoderFunc
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

	flushFunc := func(m ronykit.Message) error {
		data, err := m.Marshal()
		if err != nil {
			return err
		}

		return conn.Write(streamID, data)
	}

	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		handlers, err := r.route(rc.GetMethod(), rc.GetPath(), conn)
		if err != nil {
			return err
		}

		execFunc(r.decoder(conn, in), flushFunc, handlers...)

		return nil
	}

}

func (r rest) Gateway() ronykit.Gateway {
	return r.gw
}

func (r *rest) Dispatcher() ronykit.Dispatcher {
	return r
}
