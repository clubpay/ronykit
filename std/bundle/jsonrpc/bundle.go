package jsonrpc

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/panjf2000/gnet"
	"github.com/ronaksoft/ronykit"
)

type bundle struct {
	listen string
	eh     gnet.EventHandler
	d      ronykit.GatewayDelegate
	r      router
	ep     sync.Pool
}

func New(opts ...Option) *bundle {
	b := &bundle{
		r: router{
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
	_ = gnet.Stop(ctx, b.listen)
}

func (b *bundle) Subscribe(d ronykit.GatewayDelegate) {
	b.d = d
}

func (b *bundle) Dispatch(c ronykit.Conn, in []byte) (ronykit.DispatchFunc, error) {
	conn, ok := c.(*wsConn)
	if !ok {
		panic("BUG!! incorrect connection")
	}
	env := b.acquireEnvelope()
	err := json.Unmarshal(in, env)
	if err != nil {
		return nil, err
	}

	routeData := b.r.routes[env.Predicate]
	if routeData.Handlers == nil {
		return nil, errNoHandler
	}

	return func(ctx *ronykit.Context, execFunc ronykit.ExecuteFunc) error {
		for k, v := range env.Header {
			ctx.Set(k, v)
		}

		writeFunc := func(m ronykit.Message, _ ...string) {
			data, err := m.Marshal()
			if ctx.Error(err) {
				return
			}

			_, err = conn.Write(data)
			ctx.Error(err)
		}

		execFunc(env, writeFunc, routeData.Handlers...)

		b.releaseEnvelope(env)

		return nil
	}, nil
}

func (b *bundle) SetHandler(predicate string, handlers ...ronykit.Handler) {
	b.r.routes[predicate] = routerData{
		Handlers: handlers,
	}
}

func (b *bundle) acquireEnvelope() *Envelope {
	e, ok := b.ep.Get().(*Envelope)
	if !ok {
		e = &Envelope{}
	}

	return e
}

func (b *bundle) releaseEnvelope(e *Envelope) {
	for k := range e.Header {
		delete(e.Header, k)
	}
	e.Predicate = e.Predicate[:0]
	e.ID = 0
}

var (
	errNoHandler = fmt.Errorf("no handler for request")
)
