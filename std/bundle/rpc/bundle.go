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
	queryFactory   = "__rpc__factory"
)

type bundle struct {
	listen  string
	eh      gnet.EventHandler
	d       ronykit.GatewayDelegate
	decoder func(data []byte, e *MessageContainer) error
	mux     mux
}

func New(opts ...Option) *bundle {
	b := &bundle{
		mux: mux{
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

		factory, ok := rt.Query(queryFactory).(ronykit.MessageFactory)
		if !ok {
			panic("factory is not set in Service's Contract")
		}

		b.mux.routes[predicate] = routerData{
			ServiceName: svc.Name(),
			Handlers:    h,
			Factory:     factory,
		}
	}
}

func (b *bundle) Dispatch(c ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	_, ok := c.(*wsConn)
	if !ok {
		panic("BUG!! incorrect connection")
	}

	rpcMsgContainer := acquireMsgContainer()
	err := b.decoder(in, rpcMsgContainer)
	if err != nil {
		return nil, err
	}

	routeData := b.mux.routes[rpcMsgContainer.Predicate]
	if routeData.Handlers == nil {
		return nil, errNoHandler
	}

	msg := routeData.Factory()
	err = rpcMsgContainer.Unmarshal(msg)
	if err != nil {
		return nil, err
	}

	writeFunc := func(conn ronykit.Conn, e *ronykit.Envelope, modifiers ...ronykit.Modifier) error {
		for idx := range modifiers {
			modifiers[idx](e)
		}

		data, err := e.GetMsg().Marshal()
		if err != nil {
			return err
		}

		_, err = conn.Write(data)

		return err
	}

	// return the DispatchFunc
	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		ctx.In().
			SetHdrMap(rpcMsgContainer.Header).
			SetMsg(msg)

		ctx.
			Set(ronykit.CtxServiceName, routeData.ServiceName).
			Set(ronykit.CtxRoute, rpcMsgContainer.Predicate)

		// run the execFunc with generated params
		execFunc(writeFunc, routeData.Handlers...)

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
	err := gnet.Stop(ctx, b.listen)
	if err != nil {
		fmt.Println("got error on shutdown: ", err)
	}
}

func (b *bundle) Subscribe(d ronykit.GatewayDelegate) {
	b.d = d
}

var (
	errNoHandler = fmt.Errorf("no handler for request")
)
