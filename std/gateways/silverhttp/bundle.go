package silverhttp

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/kit/errors"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/std/gateways/silverhttp/httpmux"
	"github.com/go-www/silverlining"
	reuse "github.com/libp2p/go-reuseport"
)

const (
	queryMethod    = "silverhttp.method"
	queryPath      = "silverhttp.path"
	queryDecoder   = "silverhttp.decoder"
	queryPredicate = "silverhttp.predicate"
)

type bundle struct {
	listen  string
	l       kit.Logger
	d       kit.GatewayDelegate
	srv     *silverlining.Server
	srvName string

	connPool sync.Pool
	cors     *cors
	httpMux  *httpmux.Mux
}

var _ kit.Gateway = (*bundle)(nil)

func New(opts ...Option) (kit.Gateway, error) {
	r := &bundle{
		httpMux: &httpmux.Mux{
			RedirectTrailingSlash:  true,
			RedirectFixedPath:      true,
			HandleMethodNotAllowed: true,
			HandleOPTIONS:          true,
		},
		srv: &silverlining.Server{},
		l:   common.NewNopLogger(),
	}
	for _, opt := range opts {
		opt(r)
	}

	r.srv.Handler = r.httpHandler

	return r, nil
}

func MustNew(opts ...Option) kit.Gateway {
	b, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return b
}

func (b *bundle) Start(ctx context.Context, cfg kit.GatewayStartConfig) error {
	var (
		ln  net.Listener
		err error
	)
	if cfg.ReusePort {
		ln, err = reuse.Listen("tcp4", b.listen)
	} else {
		ln, err = net.Listen("tcp4", b.listen)
	}
	if err != nil {
		return err
	}

	go func() {

		err = b.srv.Serve(ln)
		if err != nil {
			b.l.Errorf("got error on serving: %v", err)
			panic(err)
		}
	}()

	return nil
}

func (b *bundle) Shutdown(_ context.Context) error {
	return nil
}

func (b *bundle) Register(
	svcName, contractID string, enc kit.Encoding, sel kit.RouteSelector, input kit.Message,
) {
	b.registerREST(svcName, contractID, enc, sel, input)
}

func (b *bundle) registerREST(
	svcName, contractID string, enc kit.Encoding, sel kit.RouteSelector, input kit.Message,
) {
	restSelector, ok := sel.(kit.RESTRouteSelector)
	if !ok {
		return
	}

	if restSelector.GetMethod() == "" || restSelector.GetPath() == "" {
		return
	}

	decoder, ok := restSelector.Query(queryDecoder).(DecoderFunc)
	if !ok || decoder == nil {
		decoder = reflectDecoder(enc, kit.CreateMessageFactory(input))
	}

	var methods []string
	if method := restSelector.GetMethod(); method == MethodWildcard {
		methods = append(methods,
			MethodGet, MethodPost, MethodPut, MethodPatch, MethodDelete, MethodOptions,
			MethodConnect, MethodTrace, MethodHead,
		)
	} else {
		methods = append(methods, method)
	}

	for _, method := range methods {
		b.httpMux.Handle(
			method, restSelector.GetPath(),
			&httpmux.RouteData{
				ServiceName: svcName,
				ContractID:  contractID,
				Method:      method,
				Path:        restSelector.GetPath(),
				Decoder:     decoder,
			},
		)
	}
}

func (b *bundle) Subscribe(d kit.GatewayDelegate) {
	b.d = d
}

func (b *bundle) Dispatch(ctx *kit.Context, in []byte) (kit.ExecuteArg, error) {
	switch ctx.Conn().(type) {
	case *httpConn:
		return b.httpDispatch(ctx, in)
	default:
		panic("BUG!! incorrect connection")
	}
}

func (b *bundle) httpHandler(ctx *silverlining.Context) {
	c, ok := b.connPool.Get().(*httpConn)
	if !ok {
		c = &httpConn{}
	}

	httpBody, err := ctx.Body()
	if err != nil {
		return
	}

	c.ctx = ctx
	b.d.OnOpen(c)
	b.d.OnMessage(c, b.httpWriteFunc, httpBody)
	b.d.OnClose(c.ConnID())

	b.connPool.Put(c)
}

func (b *bundle) httpDispatch(ctx *kit.Context, in []byte) (kit.ExecuteArg, error) {
	//nolint:forcetypeassert
	conn := ctx.Conn().(*httpConn)

	routeData, params, _ := b.httpMux.Lookup(conn.GetMethod(), conn.GetPath())

	// check CORS rules before even returning errRouteNotFound. This makes sure that
	// we handle any CORS even for non-routable requests.
	b.cors.handle(conn, routeData != nil)

	if routeData == nil {
		return kit.NoExecuteArg, kit.ErrNoHandler
	}

	// Walk over all the query params
	for _, p := range conn.ctx.QueryParams() {
		params = append(
			params,
			httpmux.Param{
				Key:   utils.B2S(p.Key),
				Value: utils.B2S(p.Value),
			},
		)
	}

	m, err := routeData.Decoder(params, in)
	if err != nil {
		return kit.NoExecuteArg, errors.Wrap(kit.ErrDecodeIncomingMessageFailed, err)
	}

	ctx.In().
		SetHdrWalker(conn).
		SetMsg(m)

	return kit.ExecuteArg{
		ServiceName: routeData.ServiceName,
		ContractID:  routeData.ContractID,
		Route:       fmt.Sprintf("%s %s", routeData.Method, routeData.Path),
	}, nil
}

func (b *bundle) httpWriteFunc(c kit.Conn, e *kit.Envelope) error {
	rc, ok := c.(*httpConn)
	if !ok {
		panic("BUG!! incorrect connection")
	}

	var (
		data []byte
		err  error
	)

	data, err = kit.MarshalMessage(e.GetMsg())
	if err != nil {
		return err
	}

	resHdr := rc.ctx.ResponseHeaders()
	e.WalkHdr(
		func(key string, val string) bool {
			resHdr.Set(key, val)

			return true
		},
	)

	_, err = rc.ctx.Write(data)

	return err
}
