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

func New(opts ...Option) (*bundle, error) {
	b := &bundle{
		routes:        map[string]*routeData{},
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

func (b *bundle) Register(
	svcName, contractID string, enc kit.Encoding, sel kit.RouteSelector, input kit.Message,
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
		return kit.NoExecuteArg, kit.ErrDecodeIncomingContainerFailed
	}

	inputMsgContainer := b.rpcInFactory()
	err := inputMsgContainer.Unmarshal(in)
	if err != nil {
		return kit.NoExecuteArg, errors.Wrap(kit.ErrDecodeIncomingMessageFailed, err)
	}

	routeData := b.routes[inputMsgContainer.GetHdr(b.predicateKey)]
	if routeData == nil {
		return kit.NoExecuteArg, kit.ErrNoHandler
	}

	msg := routeData.Factory()
	err = inputMsgContainer.ExtractMessage(msg)
	if err != nil {
		return kit.NoExecuteArg, errors.Wrap(kit.ErrDecodeIncomingMessageFailed, err)
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

func (b *bundle) Start(_ context.Context) error {
	go func() {
		opts := []gnet.Option{
			gnet.WithMulticore(true),
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
