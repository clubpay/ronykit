package jsonrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/goccy/go-json"
	"github.com/panjf2000/gnet"
	"github.com/ronaksoft/ronykit"
)

type bundle struct {
	listen string
	eh     gnet.EventHandler
	d      ronykit.GatewayDelegate
	r      router
}

func New(opts ...Option) *bundle {
	b := &bundle{
		r: router{
			routes: map[string]routerData{},
		},
	}
	b.eh = &gateway{
		b:     b,
		conns: map[uint64]*wsConn{},
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

func (b *bundle) Start() {
	go func() {
		err := gnet.Serve(b.eh, b.listen,
			gnet.WithMulticore(true),
		)
		if err != nil {
			panic(err)
		}
	}()
}

func (b *bundle) Shutdown() {
	ctx, cf := context.WithTimeout(context.Background(), time.Minute)
	defer cf()
	_ = gnet.Stop(ctx, b.listen)
}

func (b *bundle) Subscribe(d ronykit.GatewayDelegate) {
	b.d = d
}

func (b *bundle) Dispatch(conn ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	env := &Envelope{}
	err := json.Unmarshal(in, env)
	if err != nil {
		return nil, err
	}

	routeData := b.r.routes[env.Predicate]
	if routeData.Handlers == nil {
		return nil, errNoHandler
	}

	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		writeFunc := func(m ronykit.Message) {
			data, err := m.Marshal()
			if err != nil {
				ctx.Error(err)

				return
			}

			_, err = conn.Write(data)
			ctx.Error(err)
		}

		execFunc(env, writeFunc, routeData.Handlers...)

		return nil
	}, nil
}

func (b *bundle) SetHandler(predicate string, handlers ...ronykit.Handler) {
	b.r.routes[predicate] = routerData{
		Handlers: handlers,
	}
}

var (
	errNoHandler = fmt.Errorf("no handler for request")
)
