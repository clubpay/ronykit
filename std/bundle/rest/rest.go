package rest

import (
	"fmt"
	"github.com/ronaksoft/ronykit"
)

var (
	ErrConnectionIsNoREST = fmt.Errorf("connection is not REST")
	ErrRouteNotFound      = fmt.Errorf("route not found")
)

type rest struct {
	routes map[string]*trie
}

func New() *rest {
	r := &rest{
		routes: map[string]*trie{},
	}

	return r
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

		// TODO:: we need to extract 'm' from 'in'. Or just pass byte slice ?!!
		execFunc(nil, flushFunc, handlers...)

		return nil
	}

}
