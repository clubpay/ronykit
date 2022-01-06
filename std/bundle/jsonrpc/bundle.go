package jsonrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/goccy/go-json"
	"github.com/panjf2000/gnet"
	"github.com/ronaksoft/ronykit"
)

const (
	queryPredicate = "predicate"
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

func (b *bundle) Dispatch(c ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	conn, ok := c.(*wsConn)
	if !ok {
		panic("BUG!! incorrect connection")
	}
	env := acquireEnvelope()
	err := json.Unmarshal(in, env)
	if err != nil {
		return nil, err
	}

	routeData := b.r.routes[env.Predicate]
	if routeData.Handlers == nil {
		return nil, errNoHandler
	}

	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		for k, v := range env.Header {
			ctx.Set(k, v)
		}

		ctx.Set(ronykit.CtxServiceName, routeData.ServiceName)
		ctx.Set(ronykit.CtxRoute, env.Predicate)

		writeFunc := func(m ronykit.Message, _ ...string) error {
			data, err := m.Marshal()
			if err != nil {
				return err
			}

			_, err = conn.Write(data)

			return err
		}

		execFunc(env, writeFunc, routeData.Handlers...)

		releaseEnvelope(env)

		return nil
	}, nil
}

func (b *bundle) Register(svc ronykit.Service) {
	for _, rt := range svc.Routes() {
		var h []ronykit.Handler
		h = append(h, svc.PreHandlers()...)
		h = append(h, rt.Handlers()...)
		h = append(h, svc.PostHandlers()...)

		predicate, ok := rt.Query(queryPredicate).(string)
		if !ok {
			panic("predicate is not set in Service's Route")
		}

		b.r.routes[predicate] = routerData{
			ServiceName: svc.Name(),
			Handlers:    h,
		}
	}
}

var (
	errNoHandler = fmt.Errorf("no handler for request")
)
