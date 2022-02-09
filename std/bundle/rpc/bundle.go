package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/goccy/go-json"
	"github.com/panjf2000/gnet"
	log "github.com/ronaksoft/golog"
	"github.com/ronaksoft/ronykit"
)

const (
	queryPredicate = "__rpc__predicate"
	queryFactory   = "__rpc__factory"
)

type bundle struct {
	listen       string
	l            log.Logger
	eh           gnet.EventHandler
	d            ronykit.GatewayDelegate
	predicateKey string
	mux          mux
}

func New(opts ...Option) *bundle {
	b := &bundle{
		mux: mux{
			routes: map[string]routerData{},
		},
		predicateKey: "predicate",
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
			continue
		}

		factory, ok := rt.Query(queryFactory).(ronykit.MessageFactory)
		if !ok {
			continue
		}

		b.mux.routes[predicate] = routerData{
			ServiceName: svc.Name(),
			Predicate:   predicate,
			Handlers:    h,
			Modifiers:   rt.Modifiers(),
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
	err := json.Unmarshal(in, rpcMsgContainer)
	if err != nil {
		return nil, err
	}

	routeData := b.mux.routes[rpcMsgContainer.Header[b.predicateKey]]
	if routeData.Handlers == nil {
		return nil, errNoHandler
	}

	msg := routeData.Factory()
	err = rpcMsgContainer.Unmarshal(msg)
	if err != nil {
		return nil, err
	}

	writeFunc := func(conn ronykit.Conn, e *ronykit.Envelope) error {
		for idx := range routeData.Modifiers {
			routeData.Modifiers[idx](e)
		}

		rpcMsgContainer := acquireMsgContainer()
		e.WalkHdr(func(key string, val string) bool {
			rpcMsgContainer.Header[key] = val

			return true
		})
		rpcMsgContainer.Payload, err = e.GetMsg().Marshal()
		if err != nil {
			return err
		}

		data, err := rpcMsgContainer.Marshal()
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
			Set(ronykit.CtxRoute, routeData.Predicate)

		// run the execFunc with generated params
		execFunc(writeFunc, routeData.Handlers...)

		return nil
	}, nil
}

func (b *bundle) Start() {
	go func() {
		opts := []gnet.Option{
			gnet.WithMulticore(true),
		}
		if b.l != nil {
			opts = append(opts, gnet.WithLogger(b.l.Sugared()))
		}
		err := gnet.Serve(b.eh, b.listen, opts...)
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
