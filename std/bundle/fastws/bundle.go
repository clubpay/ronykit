package fastws

import (
	"context"
	"fmt"
	"time"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/common"
	"github.com/goccy/go-json"
	"github.com/panjf2000/gnet"
)

const (
	queryPredicate = "fastws.predicate"
)

type bundle struct {
	listen string
	l      ronykit.Logger
	eh     gnet.EventHandler
	d      ronykit.GatewayDelegate
	enc    ronykit.Encoding

	predicateKey  string
	routes        map[string]routeData
	rpcInFactory  ronykit.IncomingRPCFactory
	rpcOutFactory ronykit.OutgoingRPCFactory
}

func New(opts ...Option) (*bundle, error) {
	b := &bundle{
		routes:       map[string]routeData{},
		predicateKey: "predicate",
		rpcInFactory: func() ronykit.IncomingRPCContainer {
			return &common.SimpleIncomingJSONRPC{}
		},
		rpcOutFactory: func() ronykit.OutgoingRPCContainer {
			return &common.SimpleOutgoingJSONRPC{}
		},
	}
	gw, err := newGateway(b)
	if err != nil {
		return nil, err
	}

	b.eh = gw

	for _, opt := range opts {
		opt(b)
	}

	return b, nil
}

func MustNew(opts ...Option) *bundle {
	b, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return b
}

func (b *bundle) Register(svc ronykit.Service) {
	for _, contract := range svc.Contracts() {
		var h []ronykit.HandlerFunc
		h = append(h, svc.PreHandlers()...)
		h = append(h, contract.Handlers()...)
		h = append(h, svc.PostHandlers()...)

		rpcSelector, ok := contract.Selector().(ronykit.RPCRouteSelector)
		if !ok {
			continue
		}

		b.routes[rpcSelector.GetPredicate()] = routeData{
			ServiceName: svc.Name(),
			Predicate:   rpcSelector.GetPredicate(),
			Handlers:    h,
			Modifiers:   contract.Modifiers(),
			Factory:     ronykit.CreateMessageFactory(contract.Input()),
		}
	}
}

func (b *bundle) Dispatch(c ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	if _, ok := c.(*wsConn); !ok {
		panic("BUG!! incorrect connection")
	}

	inputMsgContainer := b.rpcInFactory()
	err := json.Unmarshal(in, inputMsgContainer)
	if err != nil {
		return nil, err
	}

	routeData := b.routes[inputMsgContainer.GetHdr(b.predicateKey)]
	if routeData.Handlers == nil {
		return nil, errNoHandler
	}

	msg := routeData.Factory()
	err = inputMsgContainer.Fill(msg)
	if err != nil {
		return nil, err
	}

	writeFunc := func(conn ronykit.Conn, e *ronykit.Envelope) error {
		outputMsgContainer := b.rpcOutFactory()
		outputMsgContainer.SetPayload(e.GetMsg())
		e.WalkHdr(func(key string, val string) bool {
			outputMsgContainer.SetHdr(key, val)

			return true
		})

		data, err := outputMsgContainer.Marshal()
		if err != nil {
			return err
		}

		_, err = conn.Write(data)

		return err
	}

	// return the DispatchFunc
	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		ctx.In().
			SetHdrMap(inputMsgContainer.GetHdrMap()).
			SetMsg(msg)

		ctx.
			Set(ronykit.CtxServiceName, routeData.ServiceName).
			Set(ronykit.CtxRoute, routeData.Predicate)

		ctx.AddModifier(routeData.Modifiers...)

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
		b.l.Error("got error on shutdown: ", err)
	}
}

func (b *bundle) Subscribe(d ronykit.GatewayDelegate) {
	b.d = d
}

var errNoHandler = fmt.Errorf("no handler for request")
