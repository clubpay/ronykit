package fastws

import (
	"context"
	"fmt"
	"time"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/internal/common"
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

	predicateKey  string
	routes        map[string]routeData
	rpcInFactory  ronykit.IncomingRPCFactory
	rpcOutFactory ronykit.OutgoingRPCFactory
}

var _ ronykit.Gateway = (*bundle)(nil)

func New(opts ...Option) (*bundle, error) {
	b := &bundle{
		routes:        map[string]routeData{},
		predicateKey:  "predicate",
		rpcInFactory:  common.SimpleIncomingJSONRPC,
		rpcOutFactory: common.SimpleOutgoingJSONRPC,
		l:             common.NewNopLogger(),
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
		rpcSelector, ok := contract.RouteSelector().(ronykit.RPCRouteSelector)
		if !ok {
			continue
		}

		b.routes[rpcSelector.GetPredicate()] = routeData{
			ServiceName: svc.Name(),
			Predicate:   rpcSelector.GetPredicate(),
			Handlers:    contract.Handlers(),
			Modifiers:   contract.Modifiers(),
			Factory:     ronykit.CreateMessageFactory(contract.Input()),
		}
	}
}

func (b *bundle) Dispatch(ctx *ronykit.Context, in []byte, execFunc ronykit.ExecuteFunc) error {
	inputMsgContainer := b.rpcInFactory()
	err := json.Unmarshal(in, inputMsgContainer)
	if err != nil {
		return err
	}

	routeData := b.routes[inputMsgContainer.GetHdr(b.predicateKey)]
	if routeData.Handlers == nil {
		return errNoHandler
	}

	msg := routeData.Factory()
	err = inputMsgContainer.Fill(msg)
	if err != nil {
		return err
	}

	ctx.In().
		SetHdrMap(inputMsgContainer.GetHdrMap()).
		SetMsg(msg)

	ctx.
		Set(ronykit.CtxServiceName, routeData.ServiceName).
		Set(ronykit.CtxRoute, routeData.Predicate)

	ctx.AddModifier(routeData.Modifiers...)

	// run the execFunc with generated params
	execFunc(
		ronykit.ExecuteArg{
			WriteFunc:        b.writeFunc,
			HandlerFuncChain: routeData.Handlers,
		},
	)

	return nil
}

func (b *bundle) writeFunc(conn ronykit.Conn, e *ronykit.Envelope) error {
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

func (b *bundle) Start(_ context.Context) error {
	go func() {
		opts := []gnet.Option{
			gnet.WithMulticore(true),
		}
		err := gnet.Serve(b.eh, b.listen, opts...)
		if err != nil {
			panic(err)
		}
	}()

	return nil
}

func (b *bundle) Shutdown(ctx context.Context) error {
	ctx, cf := context.WithTimeout(ctx, time.Minute)
	defer cf()
	err := gnet.Stop(ctx, b.listen)
	if err != nil {
		return err
	}

	return nil
}

func (b *bundle) Subscribe(d ronykit.GatewayDelegate) {
	b.d = d
}

var errNoHandler = fmt.Errorf("no handler for request")
