package jsonrpc

import (
	"context"
	"github.com/goccy/go-json"
	"github.com/panjf2000/gnet"
	"github.com/ronaksoft/ronykit"
	"time"
)

type MessageFactory func() ronykit.Message

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
		conns: map[uint64]*connWrap{},
	}

	for _, opt := range opts {
		opt(b)
	}

	return b
}

func (b *bundle) Start() {
	go func() {
		_ = gnet.Serve(b.eh, b.listen,
			gnet.WithMulticore(true),
		)
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

func (b *bundle) Dispatch(conn ronykit.Conn, streamID int64, in []byte) ronykit.DispatchFunc {
	env := &Envelope{}
	err := json.Unmarshal(in, env)
	if err != nil {
		return nil
	}

	routeData := b.r.routes[env.Predicate]
	if routeData.Handlers == nil {
		return nil
	}

	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		writeFunc := func(m ronykit.Message) {
			data, err := m.Marshal()
			if err != nil {
				ctx.Error(err)

				return
			}
			ctx.Error(conn.Write(streamID, data))
		}

		execFunc(routeData.RequestFactory(), writeFunc, routeData.Handlers...)
		
		return nil
	}

}

func (b *bundle) SetHandler(predicate string, factory MessageFactory, handlers ...ronykit.Handler) {
	b.r.routes[predicate] = routerData{
		RequestFactory: factory,
		Handlers:       handlers,
	}
}
