package fastws

import (
	"context"
	"time"

	"github.com/clubpay/ronykit"
	"github.com/clubpay/ronykit/internal/common"
	"github.com/clubpay/ronykit/internal/errors"
	"github.com/panjf2000/gnet/v2"
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
	routes        map[string]*routeData
	rpcInFactory  ronykit.IncomingRPCFactory
	rpcOutFactory ronykit.OutgoingRPCFactory
}

var _ ronykit.Gateway = (*bundle)(nil)

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
	svcName, contractID string, enc ronykit.Encoding, sel ronykit.RouteSelector, input ronykit.Message,
) {
	rpcSelector, ok := sel.(ronykit.RPCRouteSelector)
	if !ok {
		return
	}

	b.routes[rpcSelector.GetPredicate()] = &routeData{
		ServiceName: svcName,
		ContractID:  contractID,
		Predicate:   rpcSelector.GetPredicate(),
		Factory:     ronykit.CreateMessageFactory(input),
	}
}

func (b *bundle) Dispatch(ctx *ronykit.Context, in []byte) (ronykit.ExecuteArg, error) {
	if len(in) == 0 {
		return ronykit.NoExecuteArg, ronykit.ErrDecodeIncomingContainerFailed
	}

	inputMsgContainer := b.rpcInFactory()
	err := inputMsgContainer.Unmarshal(in)
	if err != nil {
		return ronykit.NoExecuteArg, errors.Wrap(ronykit.ErrDecodeIncomingMessageFailed, err)
	}

	routeData := b.routes[inputMsgContainer.GetHdr(b.predicateKey)]
	if routeData == nil {
		return ronykit.NoExecuteArg, ronykit.ErrNoHandler
	}

	msg := routeData.Factory()
	err = inputMsgContainer.ExtractMessage(msg)
	if err != nil {
		return ronykit.NoExecuteArg, errors.Wrap(ronykit.ErrDecodeIncomingMessageFailed, err)
	}

	ctx.In().
		SetID(inputMsgContainer.GetID()).
		SetHdrMap(inputMsgContainer.GetHdrMap()).
		SetMsg(msg)

	// release the container
	inputMsgContainer.Release()

	return ronykit.ExecuteArg{
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

func (b *bundle) Subscribe(d ronykit.GatewayDelegate) {
	b.d = d
}
