package kit

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/clubpay/ronykit/kit/errors"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/buf"
)

type Cluster interface {
	// Start starts the gateway to accept connections.
	Start(ctx context.Context) error
	// Shutdown shuts down the gateway gracefully.
	Shutdown(ctx context.Context) error
	// Subscribe registers the southBridge as the delegate of the backend.
	Subscribe(id string, d ClusterDelegate)
	// Publish sends the data to the other instance identified by id.
	Publish(id string, data []byte) error
	// Subscribers return the list of the instance ids.
	Subscribers() ([]string, error)
}

type ClusterWithStore interface {
	// Store returns a shared key-value store between all instances.
	Store() ClusterStore
}

type ClusterDelegate interface {
	OnMessage(data []byte)
}

// southBridge is a container component that connects EdgeServers using Cluster.
type southBridge struct {
	ctxPool
	id string
	wg *sync.WaitGroup
	eh ErrHandlerFunc
	c  map[string]Contract
	cb Cluster
	tp TracePropagator
	l  Logger

	inProgressMtx utils.SpinLock
	inProgress    map[string]*clusterConn
	msgFactories  map[string]MessageFactoryFunc
}

var _ ClusterDelegate = (*southBridge)(nil)

func (sb *southBridge) registerContract(input, output Message) {
	sb.msgFactories[reflect.TypeOf(input).String()] = CreateMessageFactory(input)
	sb.msgFactories[reflect.TypeOf(output).String()] = CreateMessageFactory(output)
}

func (sb *southBridge) Start(ctx context.Context) error {
	return sb.cb.Start(ctx)
}

func (sb *southBridge) Shutdown(ctx context.Context) error {
	return sb.cb.Shutdown(ctx)
}

func (sb *southBridge) OnMessage(data []byte) {
	carrier := &envelopeCarrier{}
	if err := carrier.FromJSON(data); err != nil {
		sb.eh(nil, fmt.Errorf("failed to decode envelope carrier: %w", err))

		return
	}

	sb.wg.Add(1)

	switch carrier.Kind {
	case incomingCarrier:
		go sb.onIncomingMessage(carrier)
	default:
		conn := sb.getConn(carrier.SessionID)
		if conn != nil {
			ctx := sb.acquireCtx(conn)
			ctx.sb = sb
			select {
			case conn.carrierChan <- carrier:
			default:
				sb.eh(ctx, ErrWritingToClusterConnection)
			}
			sb.releaseCtx(ctx)
		}
	}

	sb.wg.Done()
}

func (sb *southBridge) createSenderConn(
	carrier *envelopeCarrier, timeout time.Duration, callbackFn func(*envelopeCarrier),
) *clusterConn {
	ctx, cancelFn := context.WithCancel(context.Background())
	if timeout > 0 {
		ctx, cancelFn = context.WithTimeout(ctx, timeout)
	}

	conn := &clusterConn{
		ctx:         ctx,
		cf:          cancelFn,
		callbackFn:  callbackFn,
		cluster:     sb.cb,
		originID:    carrier.OriginID,
		sessionID:   carrier.SessionID,
		serverID:    sb.id,
		kv:          map[string]string{},
		wf:          sb.writeFunc,
		carrierChan: make(chan *envelopeCarrier, 32),
	}

	sb.inProgressMtx.Lock()
	sb.inProgress[carrier.SessionID] = conn
	sb.inProgressMtx.Unlock()

	go func(c *clusterConn) {
		for {
			select {
			case <-c.ctx.Done():
				return
			case carrier, ok := <-c.carrierChan:
				if !ok {
					c.cf()

					return
				}
				switch carrier.Kind {
				default:
					panic("invalid carrier kind")
				case outgoingCarrier:
					c.callbackFn(carrier)
				case eofCarrier:
					sb.inProgressMtx.Lock()
					delete(sb.inProgress, c.sessionID)
					sb.inProgressMtx.Unlock()

					close(c.carrierChan)
				}
			}
		}
	}(conn)

	return conn
}

func (sb *southBridge) createTargetConn(
	carrier *envelopeCarrier,
) *clusterConn {
	conn := &clusterConn{
		cluster:   sb.cb,
		originID:  carrier.OriginID,
		sessionID: carrier.SessionID,
		serverID:  sb.id,
		kv:        map[string]string{},
		wf:        sb.writeFunc,
	}

	return conn
}

func (sb *southBridge) getConn(sessionID string) *clusterConn {
	sb.inProgressMtx.Lock()
	conn := sb.inProgress[sessionID]
	sb.inProgressMtx.Unlock()

	return conn
}

func (sb *southBridge) onIncomingMessage(carrier *envelopeCarrier) {
	conn := sb.createTargetConn(carrier)
	ctx := sb.acquireCtx(conn)
	ctx.sb = sb
	ctx.forwarded = true

	msg := sb.msgFactories[carrier.Data.MsgType]()
	switch v := msg.(type) {
	case RawMessage:
		v.CopyFrom(carrier.Data.Msg)
		msg = v
	default:
		unmarshalEnvelopeCarrier(carrier.Data.Msg, msg)
	}

	if sb.tp != nil {
		ctx.SetUserContext(sb.tp.Extract(ctx.Context(), carrier.Data))
	}

	for k, v := range carrier.Data.ConnHdr {
		ctx.Conn().Set(k, v)
	}

	ctx.in.
		SetID(carrier.Data.EnvelopeID).
		SetHdrMap(carrier.Data.Hdr).
		SetMsg(msg)

	ctx.execute(
		ExecuteArg{
			ServiceName: carrier.Data.ServiceName,
			ContractID:  carrier.Data.ContractID,
			Route:       carrier.Data.Route,
		},
		sb.c[carrier.Data.ContractID],
	)

	ec := newEnvelopeCarrier(
		eofCarrier,
		carrier.SessionID,
		sb.id,
		carrier.OriginID,
	)

	ecBuf := buf.GetCap(CodeDefaultBufferSize)
	err := defaultMessageCodec.Encode(ec, ecBuf)
	if err != nil {
		sb.eh(ctx, err)
	} else if err = sb.cb.Publish(carrier.OriginID, *ecBuf.Bytes()); err != nil {
		sb.eh(ctx, err)
	}

	ecBuf.Release()
	sb.releaseCtx(ctx)
}

func (sb *southBridge) sendMessage(carrier *envelopeCarrier) error {
	ecBuf := buf.GetCap(CodeDefaultBufferSize)
	err := defaultMessageCodec.Encode(carrier, ecBuf)
	if err == nil {
		err = sb.cb.Publish(carrier.TargetID, *ecBuf.Bytes())
	}
	if err != nil {
		sb.inProgressMtx.Lock()
		delete(sb.inProgress, carrier.SessionID)
		sb.inProgressMtx.Unlock()
	}

	ecBuf.Release()

	return err
}

func (sb *southBridge) wrapWithCoordinator(c Contract) Contract {
	if c.EdgeSelector() == nil || sb == nil {
		return c
	}

	cw := &contractWrap{
		Contract: c,
		h: []HandlerFunc{
			sb.genForwarderHandler(c.EdgeSelector()),
		},
	}

	return cw
}

func (sb *southBridge) genForwarderHandler(sel EdgeSelectorFunc) HandlerFunc {
	return func(ctx *Context) {
		// if it is already a forwarded request, we should forward it again.
		// This is required to avoid infinite loops.
		if ctx.forwarded {
			return
		}

		target, err := sel(ctx.Limited())
		if ctx.Error(err) {
			return
		}

		if target == "" || target == sb.id {
			return
		}

		carrier := newEnvelopeCarrier(
			incomingCarrier,
			utils.RandomID(32),
			ctx.sb.id,
			target,
		).FillWithContext(ctx)

		err = ctx.sb.sendMessage(carrier)
		if err != nil {
			ctx.Error(err)
			ctx.StopExecution()

			return
		}

		conn := sb.createSenderConn(carrier, ctx.rxt, sb.genCallback(ctx))
		select {
		case <-conn.Done():
			ctx.Error(conn.Err())
		case <-ctx.ctx.Done():
			ctx.Error(ctx.ctx.Err())
			conn.cf()
		}

		// We should stop executing next handlers, since our request has been executed on
		// a remote machine
		ctx.StopExecution()
	}
}

func (sb *southBridge) genCallback(ctx *Context) func(carrier *envelopeCarrier) {
	return func(carrier *envelopeCarrier) {
		if carrier.Data == nil {
			return
		}
		f, ok := sb.msgFactories[carrier.Data.MsgType]
		if !ok {
			return
		}

		msg := f()
		switch msg.(type) {
		case RawMessage:
			msg = RawMessage(carrier.Data.Msg)
		default:
			unmarshalEnvelopeCarrier(carrier.Data.Msg, msg)
		}

		for k, v := range carrier.Data.ConnHdr {
			ctx.Conn().Set(k, v)
		}

		ctx.Out().
			SetID(carrier.Data.EnvelopeID).
			SetHdrMap(carrier.Data.Hdr).
			SetMsg(msg).
			Send()
	}
}

func (sb *southBridge) writeFunc(c *clusterConn, e *Envelope) error {
	ec := newEnvelopeCarrier(
		outgoingCarrier,
		c.sessionID,
		c.serverID,
		c.originID,
	).
		FillWithEnvelope(e)

	if sb.tp != nil {
		sb.tp.Inject(e.ctx.ctx, ec.Data)
	}

	ecBuf := buf.GetCap(CodeDefaultBufferSize)
	err := defaultMessageCodec.Encode(ec, ecBuf)
	if err != nil {
		return err
	}

	err = c.cluster.Publish(c.originID, *ecBuf.Bytes())
	if err != nil {
		return err
	}

	// NOTE: for extra safety will only re-use if there was no error
	ecBuf.Release()

	return nil
}

var _ Conn = (*clusterConn)(nil)

type clusterConn struct {
	clientIP string
	stream   bool
	kvMtx    sync.Mutex
	kv       map[string]string

	// target
	serverID  string
	sessionID string
	originID  string
	wf        func(c *clusterConn, e *Envelope) error
	cluster   Cluster

	// sender
	ctx         context.Context
	cf          context.CancelFunc
	callbackFn  func(carrier *envelopeCarrier)
	carrierChan chan *envelopeCarrier
}

func (c *clusterConn) ConnID() uint64 {
	return 0
}

func (c *clusterConn) ClientIP() string {
	return c.clientIP
}

func (c *clusterConn) WriteEnvelope(e *Envelope) error {
	return c.wf(c, e)
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

func (c *clusterConn) Keys() []string {
	keys := make([]string, 0, len(c.kv))
	c.kvMtx.Lock()
	for k := range c.kv {
		keys = append(keys, k)
	}
	c.kvMtx.Unlock()

	return keys
}

func (c *clusterConn) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *clusterConn) Err() error {
	return c.ctx.Err()
}

func (c *clusterConn) Cancel() {
	if c.cf != nil {
		c.cf()
	}
}

var (
	ErrSouthBridgeDisabled        = errors.New("south bridge is disabled")
	ErrWritingToClusterConnection = errors.New("writing to cluster connection is not possible")
)
