package fastws

import (
	"context"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/kit/errors"
	"github.com/panjf2000/gnet/v2"
)

const (
	queryPredicate = "fastws.predicate"
)

var noExecuteArg = kit.ExecuteArg{}

type bundle struct {
	listen string
	l      kit.Logger
	eh     gnet.EventHandler
	d      kit.GatewayDelegate

	predicateKey  string
	routes        map[string]*routeData
	rpcInFactory  kit.IncomingRPCFactory
	rpcOutFactory kit.OutgoingRPCFactory
}

var _ kit.Gateway = (*bundle)(nil)

func New(opts ...Option) (kit.Gateway, error) {
	b := &bundle{
		routes:        map[string]*routeData{},
		predicateKey:  "predicate",
		rpcInFactory:  common.SimpleIncomingJSONRPC,
		rpcOutFactory: common.SimpleOutgoingJSONRPC,
		l:             common.NewNopLogger(),
	}
	b.eh = newGateway(b)

	for _, opt := range opts {
		opt(b)
	}

	return b, nil
}

func MustNew(opts ...Option) kit.Gateway {
	b, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return b
}

func (b *bundle) Register(
	svcName, contractID string, _ kit.Encoding, sel kit.RouteSelector, input kit.Message,
) {
	rpcSelector, ok := sel.(kit.RPCRouteSelector)
	if !ok {
		return
	}

	b.routes[rpcSelector.GetPredicate()] = &routeData{
		ServiceName: svcName,
		ContractID:  contractID,
		Predicate:   rpcSelector.GetPredicate(),
		Factory:     kit.CreateMessageFactory(input),
	}
}

func (b *bundle) Dispatch(ctx *kit.Context, in []byte) (kit.ExecuteArg, error) {
	if len(in) == 0 {
		return noExecuteArg, kit.ErrDecodeIncomingContainerFailed
	}

	inputMsgContainer := b.rpcInFactory()
	err := inputMsgContainer.Unmarshal(in)
	if err != nil {
		return noExecuteArg, errors.Wrap(kit.ErrDecodeIncomingMessageFailed, err)
	}

	routeData := b.routes[inputMsgContainer.GetHdr(b.predicateKey)]
	if routeData == nil {
		return noExecuteArg, kit.ErrNoHandler
	}

	msg := routeData.Factory()
	switch v := msg.(type) {
	case kit.RawMessage:
		err = inputMsgContainer.ExtractMessage(&v)
		msg = v
	default:
		err = inputMsgContainer.ExtractMessage(msg)
	}
	if err != nil {
		return noExecuteArg, errors.Wrap(kit.ErrDecodeIncomingMessageFailed, err)
	}

	ctx.In().
		SetID(inputMsgContainer.GetID()).
		SetHdrMap(inputMsgContainer.GetHdrMap()).
		SetMsg(msg)

	// release the container
	inputMsgContainer.Release()

	return kit.ExecuteArg{
		ServiceName: routeData.ServiceName,
		ContractID:  routeData.ContractID,
		Route:       routeData.Predicate,
	}, nil
}

func (b *bundle) Start(_ context.Context, cfg kit.GatewayStartConfig) error {
	go func() {
		opts := []gnet.Option{
			gnet.WithMulticore(true),
			gnet.WithReusePort(cfg.ReusePort),
		}

		err := gnet.Run(b.eh, b.listen, opts...)
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

func (b *bundle) Subscribe(d kit.GatewayDelegate) {
	b.d = d
}
