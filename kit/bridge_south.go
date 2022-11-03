package kit

import (
	"context"
	"sync"

	"github.com/clubpay/ronykit/kit/errors"
	"github.com/clubpay/ronykit/kit/utils"
)

type ClusterBackend interface {
	// Start starts the gateway to accept connections.
	Start(ctx context.Context) error
	// Shutdown shuts down the gateway gracefully.
	Shutdown(ctx context.Context) error
	// Subscribe registers the southBridge as the delegate of the backend.
	Subscribe(id string, d ClusterDelegate)
	// Publish sends the data to the other instance identified by id.
	Publish(id string, data []byte) error
	// SetKey creates/updates a key-value pair in the cluster
	SetKey(key, value string) error
	// DeleteKey deletes the key-value pair from the cluster
	DeleteKey(key string) error
}

type ClusterDelegate interface {
	OnMessage(data []byte) error
}

// southBridge is a container component that connects EdgeServers using ClusterBackend.
type southBridge struct {
	ctxPool
	id string
	e  *EdgeServer
	cb ClusterBackend

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

func (sb *southBridge) OnMessage(data []byte) error {
	carrier := &envelopeCarrier{}
	if err := carrier.FromJSON(data); err != nil {
		return err
	}

	sb.e.wg.Add(1)
	conn := &clusterConn{
		cb:        sb.cb,
		originID:  carrier.OriginID,
		sessionID: carrier.SessionID,
		serverID:  sb.id,
	}
	ctx := sb.acquireCtx(conn)
	ctx.wf = writeFunc
	ctx.sb = sb

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

	return nil
}

func (sb *southBridge) onIncomingMessage(ctx *Context, carrier *envelopeCarrier) {
	ctx.in.
		SetID(carrier.Data.EnvelopeID).
		SetHdrMap(carrier.Data.Hdr).
		SetMsg(carrier.Data.Msg)

	ctx.handlerIndex = carrier.Data.ExecIndex
	ctx.execute(
		ExecuteArg{
			ServiceName: carrier.Data.ServiceName,
			ContractID:  carrier.Data.ContractID,
			Route:       carrier.Data.Route,
		},
		sb.e.getContract(carrier.Data.ContractID),
	)

	eof := newEnvelopeCarrier(eofCarrier, carrier.SessionID, sb.id, carrier.OriginID)
	err := sb.cb.Publish(
		carrier.OriginID,
		eof.ToJSON(),
	)
	if err != nil {
		sb.e.eh(ctx, err)
	}
}

func (sb *southBridge) onOutgoingMessage(carrier *envelopeCarrier) {
	sb.inProgressMtx.Lock()
	ch, ok := sb.inProgress[carrier.SessionID]
	sb.inProgressMtx.Unlock()
	if ok {
		ch <- carrier
	}
}

func (sb *southBridge) onEOF(carrier *envelopeCarrier) {
	sb.inProgressMtx.Lock()
	ch, ok := sb.inProgress[carrier.SessionID]
	delete(sb.inProgress, carrier.SessionID)
	sb.inProgressMtx.Unlock()
	if ok {
		close(ch)
	}
}

func (sb *southBridge) sendMessage(sessionID string, targetID string, data []byte) (<-chan *envelopeCarrier, error) {
	ch := make(chan *envelopeCarrier, 4)
	sb.inProgressMtx.Lock()
	sb.inProgress[sessionID] = ch
	sb.inProgressMtx.Unlock()

	err := sb.cb.Publish(targetID, data)
	if err != nil {
		sb.inProgressMtx.Lock()
		delete(sb.inProgress, sessionID)
		sb.inProgressMtx.Unlock()

		return nil, err
	}

	return ch, nil
}

type clusterConn struct {
	sessionID string
	originID  string
	serverID  string
	cb        ClusterBackend

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
	return len(data), c.cb.Publish(c.originID, data)
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
	c := conn.(*clusterConn) //nolint:forcetypeassert

	ec := newEnvelopeCarrier(outgoingCarrier, c.sessionID, c.serverID, c.originID)
	ec.Data = &carrierData{
		Msg:         e.GetMsg(),
		ContractID:  e.ctx.ContractID(),
		ServiceName: e.ctx.ServiceName(),
		ExecIndex:   0,
		Route:       e.ctx.Route(),
	}

	return c.cb.Publish(c.originID, ec.ToJSON())
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
		defer ctx.StopExecution()

		target, err := sel(ctx.Limited())
		if ctx.Error(err) {
			return
		}

		arg := executeRemoteArg{
			Target: target,
			In: newEnvelopeCarrier(
				incomingCarrier,
				utils.RandomID(32),
				ctx.sb.id,
				target,
			).FillWithContext(ctx),
			OutCallback: func(carrier *envelopeCarrier) {
				ctx.Out().
					SetID(carrier.Data.EnvelopeID).
					SetHdrMap(carrier.Data.Hdr).
					SetMsg(carrier.Data.Msg).
					Send()
			},
		}

		err = ctx.executeRemote(arg)
		ctx.Error(err)
	}
}

var ErrSouthBridgeDisabled = errors.New("south bridge is disabled")
