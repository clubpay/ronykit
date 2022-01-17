package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/panjf2000/gnet"
	"github.com/ronaksoft/ronykit"
)

const (
	queryPredicate = "__rpc__predicate"
)

type bundle struct {
	listen string
	eh     gnet.EventHandler
	d      ronykit.GatewayDelegate
	df     DecoderFunc
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

func (b *bundle) Register(svc ronykit.Service) {
	for _, rt := range svc.Contracts() {
		var h []ronykit.Handler
		h = append(h, svc.PreHandlers()...)
		h = append(h, rt.Handlers()...)
		h = append(h, svc.PostHandlers()...)

		predicate, ok := rt.Query(queryPredicate).(string)
		if !ok {
			panic("predicate is not set in Service's Contract")
		}

		b.r.routes[predicate] = routerData{
			ServiceName: svc.Name(),
			Handlers:    h,
		}
	}
}

func (b *bundle) Dispatch(c ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	conn, ok := c.(*wsConn)
	if !ok {
		panic("BUG!! incorrect connection")
	}

	rpcEnvelope := acquireEnvelope()
	err := b.df(in, rpcEnvelope)
	if err != nil {
		return nil, err
	}

	routeData := b.r.routes[rpcEnvelope.Predicate]
	if routeData.Handlers == nil {
		return nil, errNoHandler
	}

	msg := routeData.Factory()
	err = rpcEnvelope.Unmarshal(msg)
	if err != nil {
		return nil, err
	}

	env := ronykit.NewEnvelope().SetMsg(msg)
	for k, v := range rpcEnvelope.Header {
		env.SetHdr(k, v)
	}

	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		ctx.Set(ronykit.CtxServiceName, routeData.ServiceName)
		ctx.Set(ronykit.CtxRoute, rpcEnvelope.Predicate)

		writeFunc := func(e *ronykit.Envelope) error {
			data, err := e.GetMsg().Marshal()
			if err != nil {
				return err
			}

			_, err = conn.Write(data)

			return err
		}

		execFunc(env, writeFunc, routeData.Handlers...)

		env.Release()

		return nil
	}, nil
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

var (
	errNoHandler = fmt.Errorf("no handler for request")
)
