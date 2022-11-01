package ronykit

import (
	"context"
	"sync"

	"github.com/clubpay/ronykit/internal/errors"
	"github.com/clubpay/ronykit/utils"
)

type ClusterBackend interface {
	// Start starts the gateway to accept connections.
	Start(ctx context.Context) error
	// Shutdown shuts down the gateway gracefully.
	Shutdown(ctx context.Context) error
	Subscribe(id string, d ClusterDelegate)
	Publish(id string, data []byte) error
}

type ClusterDelegate interface {
	OnMessage(data []byte)
}

// southBridge is a container component that connects EdgeServer with a Cluster type Bundle.
type southBridge struct {
	ctxPool
	id string
	e  *EdgeServer
	cb ClusterBackend

	shutdownChan  chan struct{}
	inProgressMtx utils.SpinLock
	inProgress    map[string]chan *envelopeCarrier
}

var _ ClusterDelegate = (*southBridge)(nil)

func (sb *southBridge) Start(ctx context.Context) error {
	return sb.cb.Start(ctx)
}

func (sb *southBridge) Shutdown(ctx context.Context) error {
	return sb.cb.Shutdown(ctx)
}

func (sb *southBridge) OnMessage(data []byte) {
	sb.e.wg.Add(1)
	conn := &clusterConn{}
	ctx := sb.acquireCtx(conn)
	ctx.wf = writeFunc
	ctx.sb = sb

	carrier, err := envelopeCarrierFromData(data)
	if err != nil {
		sb.e.eh(ctx, errors.Wrap(ErrDispatchFailed, err))
		sb.releaseCtx(ctx)
		sb.e.wg.Done()

		return
	}

	switch carrier.Kind {
	case incomingCarrier:
		sb.onIncomingMessage(ctx, carrier)
	case outgoingCarrier:
		sb.onOutgoingMessage(carrier)
	case eofCarrier:
		sb.onEOF(carrier)
	}

	sb.releaseCtx(ctx)
	sb.e.wg.Done()
}

func (sb *southBridge) onIncomingMessage(ctx *Context, carrier *envelopeCarrier) {
	ctx.in.
		SetID(carrier.ID).
		SetHdrMap(carrier.Hdr).
		SetMsg(carrier.Msg)

	ctx.handlerIndex = carrier.ExecIndex
	ctx.execute(
		ExecuteArg{
			ServiceName: carrier.ServiceName,
			ContractID:  carrier.ContractID,
			Route:       carrier.Route,
		},
		sb.e.getContract(carrier.ContractID),
	)

	eof := &envelopeCarrier{
		ID:   carrier.ID,
		Kind: eofCarrier,
	}
	err := sb.cb.Publish(carrier.ID, eof.ToJSON())
	if err != nil {
		sb.e.eh(ctx, err)
	}
}

func (sb *southBridge) onOutgoingMessage(carrier *envelopeCarrier) {
	sb.inProgressMtx.Lock()
	ch, ok := sb.inProgress[carrier.ID]
	sb.inProgressMtx.Unlock()
	if ok {
		ch <- carrier
	}
}

func (sb *southBridge) onEOF(carrier *envelopeCarrier) {
	sb.inProgressMtx.Lock()
	ch, ok := sb.inProgress[carrier.ID]
	delete(sb.inProgress, carrier.ID)
	sb.inProgressMtx.Unlock()
	if ok {
		close(ch)
	}
}

type clusterConn struct {
	serverID string
	cb       ClusterBackend

	id       uint64
	clientIP string
	stream   bool

	kvMtx sync.Mutex
	kv    map[string]string
}

var _ Conn = (*clusterConn)(nil)

func (c *clusterConn) ConnID() uint64 {
	return c.id
}

func (c *clusterConn) ClientIP() string {
	return c.clientIP
}

func (c *clusterConn) Write(data []byte) (int, error) {
	return len(data), c.cb.Publish(c.serverID, data)
}

func (c *clusterConn) Stream() bool {
	return c.stream
}

func (c *clusterConn) Walk(f func(key string, val string) bool) {
	c.kvMtx.Lock()
	defer c.kvMtx.Unlock()

	for k, v := range c.kv {
		if !f(k, v) {
			return
		}
	}
}

func (c *clusterConn) Get(key string) string {
	c.kvMtx.Lock()
	v := c.kv[key]
	c.kvMtx.Unlock()

	return v
}

func (c *clusterConn) Set(key string, val string) {
	c.kvMtx.Lock()
	c.kv[key] = val
	c.kvMtx.Unlock()
}

func writeFunc(conn Conn, e *Envelope) error {
	ec := envelopeCarrier{
		ID:          e.GetID(),
		Kind:        outgoingCarrier,
		Msg:         e.GetMsg(),
		ContractID:  e.ctx.ContractID(),
		ServiceName: e.ctx.ServiceName(),
		ExecIndex:   0,
		Route:       e.ctx.Route(),
	}
	_, err := conn.Write(ec.ToJSON())

	return err
}

func wrapWithCoordinator(c Contract) Contract {
	if c.EdgeSelector() == nil {
		return c
	}

	cw := &contractWrap{
		Contract: c,
		h: []HandlerFunc{
			genForwarderHandler(c.EdgeSelector()),
		},
	}

	return cw
}

func genForwarderHandler(sel EdgeSelectorFunc) HandlerFunc {
	return func(ctx *Context) {
		target, err := sel(ctx.Limited())
		if ctx.Error(err) {
			return
		}

		arg := executeRemoteArg{
			ServerID: target,
			SentData: envelopeCarrierFromContext(ctx, incomingCarrier),
			Callback: func(carrier *envelopeCarrier) {
				ctx.Out().
					SetHdrMap(carrier.Hdr).
					SetMsg(carrier.Msg).
					Send()
			},
		}

		err = ctx.executeRemote(arg)
		if ctx.Error(err) {
			return
		}
	}
}

var ErrSouthBridgeDisabled = errors.New("south bridge is disabled")
