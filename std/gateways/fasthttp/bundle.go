package fasthttp

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/kit/errors"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/buf"
	"github.com/clubpay/ronykit/std/gateways/fasthttp/internal/httpmux"
	"github.com/clubpay/ronykit/std/gateways/fasthttp/proxy"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
)

const (
	queryMethod    = "fasthttp.method"
	queryPath      = "fasthttp.path"
	queryDecoder   = "fasthttp.decoder"
	queryPredicate = "fasthttp.predicate"
)

var noExecuteArg = kit.ExecuteArg{}

type bundle struct {
	l        kit.Logger
	d        kit.GatewayDelegate
	srv      *fasthttp.Server
	listen   string
	connPool sync.Pool
	cors     *cors

	reverseProxyPath string
	reverseProxy     *proxy.ReverseProxy
	httpMux          *httpmux.Mux
	compress         CompressionLevel

	wsUpgrade     websocket.FastHTTPUpgrader
	wsRoutes      map[string]*httpmux.RouteData
	wsEndpoint    string
	wsNextID      uint64
	predicateKey  string
	rpcInFactory  kit.IncomingRPCFactory
	rpcOutFactory kit.OutgoingRPCFactory
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
		compress:      CompressionLevelDefault,
		wsRoutes:      map[string]*httpmux.RouteData{},
		srv:           &fasthttp.Server{},
		rpcInFactory:  common.SimpleIncomingJSONRPC,
		rpcOutFactory: common.SimpleOutgoingJSONRPC,
		l:             common.NewNopLogger(),
	}

	r.wsUpgrade.CheckOrigin = func(ctx *fasthttp.RequestCtx) bool {
		return true
	}
	for _, opt := range opts {
		opt(r)
	}

	httpHandler := r.httpHandler
	switch r.compress {
	case CompressionLevelDefault:
		httpHandler = fasthttp.CompressHandlerBrotliLevel(
			r.httpHandler,
			fasthttp.CompressBrotliDefaultCompression,
			fasthttp.CompressDefaultCompression,
		)
	case CompressionLevelBestSpeed:
		httpHandler = fasthttp.CompressHandlerBrotliLevel(
			r.httpHandler,
			fasthttp.CompressBrotliBestSpeed,
			fasthttp.CompressBestSpeed,
		)
	case CompressionLevelBestCompression:
		httpHandler = fasthttp.CompressHandlerBrotliLevel(
			r.httpHandler,
			fasthttp.CompressBrotliBestCompression,
			fasthttp.CompressBestCompression,
		)
	}

	var muxHandlers []muxHandler
	if r.reverseProxy != nil {
		proxyPath := utils.S2B(r.reverseProxyPath)
		muxHandlers = append(
			muxHandlers,
			func(ctx *fasthttp.RequestCtx) bool {
				if bytes.EqualFold(ctx.Path(), proxyPath) {
					r.reverseProxy.ServeHTTP(ctx)

					return true
				}

				return false
			},
		)
	}
	if r.wsEndpoint != "" {
		wsEndpoint := utils.S2B(r.wsEndpoint)
		muxHandlers = append(
			muxHandlers,
			func(ctx *fasthttp.RequestCtx) bool {
				if ctx.IsGet() && bytes.EqualFold(ctx.Path(), wsEndpoint) {
					r.wsHandler(ctx)

					return true
				}

				return false
			},
		)
	}

	if len(muxHandlers) > 0 {
		r.srv.Handler = func(ctx *fasthttp.RequestCtx) {
			for _, h := range muxHandlers {
				if h(ctx) {
					return
				}
			}

			httpHandler(ctx)
		}
	} else {
		r.srv.Handler = httpHandler
	}

	return r, nil
}

func MustNew(opts ...Option) kit.Gateway {
	b, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return b
}

type muxHandler func(ctx *fasthttp.RequestCtx) bool

func (b *bundle) Register(
	svcName, contractID string, enc kit.Encoding, sel kit.RouteSelector, input kit.Message,
) {
	b.registerRPC(svcName, contractID, enc, sel, input)
	b.registerREST(svcName, contractID, enc, sel, input)
}

func (b *bundle) registerRPC(
	svcName, contractID string, _ kit.Encoding, sel kit.RouteSelector, input kit.Message,
) {
	rpcSelector, ok := sel.(kit.RPCRouteSelector)
	if !ok {
		// this selector is not an RPCRouteSelector then we return with no
		// extra action taken.
		return
	}

	// We don't accept selector with empty Predicate for the obvious reason.
	if rpcSelector.GetPredicate() == "" {
		return
	}

	rd := &httpmux.RouteData{
		ServiceName: svcName,
		ContractID:  contractID,
		Predicate:   rpcSelector.GetPredicate(),
		Factory:     kit.CreateMessageFactory(input),
	}

	b.wsRoutes[rd.Predicate] = rd
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

func (b *bundle) Dispatch(ctx *kit.Context, in []byte) (kit.ExecuteArg, error) {
	switch ctx.Conn().(type) {
	case *httpConn:
		return b.httpDispatch(ctx, in)
	case *wsConn:
		return b.wsDispatch(ctx, in)
	default:
		panic("BUG!! incorrect connection")
	}
}

func (b *bundle) wsHandler(ctx *fasthttp.RequestCtx) {
	_ = b.wsUpgrade.Upgrade(ctx,
		func(conn *websocket.Conn) {
			wsc := &wsConn{
				kv:       map[string]string{},
				id:       atomic.AddUint64(&b.wsNextID, 1),
				clientIP: conn.RemoteAddr().String(),
				c:        conn,
			}
			b.d.OnOpen(wsc)
			for {
				_, in, err := conn.ReadMessage()
				if err != nil {
					break
				}

				inBuf := buf.FromBytes(in)
				go b.wsHandlerExec(inBuf, wsc)
			}
			wsc.Close()
			b.d.OnClose(wsc.id)
		},
	)
}

func (b *bundle) wsHandlerExec(buf *buf.Bytes, wsc *wsConn) {
	b.d.OnMessage(wsc, b.writeFunc, *buf.Bytes())
	buf.Release()
}

func (b *bundle) wsDispatch(ctx *kit.Context, in []byte) (kit.ExecuteArg, error) {
	if len(in) == 0 {
		return noExecuteArg, kit.ErrDecodeIncomingContainerFailed
	}

	inputMsgContainer := b.rpcInFactory()
	err := inputMsgContainer.Unmarshal(in)
	if err != nil {
		return noExecuteArg, err
	}

	routeData := b.wsRoutes[inputMsgContainer.GetHdr(b.predicateKey)]
	if routeData == nil {
		return noExecuteArg, kit.ErrNoHandler
	}

	msg := routeData.Factory()
	err = inputMsgContainer.ExtractMessage(msg)
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

func (b *bundle) writeFunc(conn kit.Conn, e *kit.Envelope) error {
	switch c := conn.(type) {
	case *wsConn:
		outC := b.rpcOutFactory()
		outC.InjectMessage(e.GetMsg())
		outC.SetID(e.GetID())
		e.WalkHdr(
			func(key string, val string) bool {
				outC.SetHdr(key, val)

				return true
			},
		)

		data, err := outC.Marshal()
		if err != nil {
			return err
		}

		_, err = c.Write(data)
		outC.Release()

		return err
	case *httpConn:
		var (
			data []byte
			err  error
		)

		data, err = kit.MarshalMessage(e.GetMsg())
		if err != nil {
			return err
		}

		e.WalkHdr(
			func(key string, val string) bool {
				c.ctx.Response.Header.Set(key, val)

				return true
			},
		)

		c.ctx.SetBody(data)

		return nil
	default:
		panic("BUG!! incorrect connection")
	}
}

func (b *bundle) httpHandler(ctx *fasthttp.RequestCtx) {
	c, ok := b.connPool.Get().(*httpConn)
	if !ok {
		c = &httpConn{}
	}

	c.ctx = ctx
	b.d.OnOpen(c)
	b.d.OnMessage(c, b.writeFunc, ctx.PostBody())
	b.d.OnClose(c.ConnID())

	b.connPool.Put(c)
}

func (b *bundle) httpDispatch(ctx *kit.Context, in []byte) (kit.ExecuteArg, error) {
	//nolint:forcetypeassert
	conn := ctx.Conn().(*httpConn)

	routeData, params, _ := b.httpMux.Lookup(conn.GetMethod(), conn.GetPath())

	// Check CORS rules before even returning errRouteNotFound.
	// This makes sure that we handle any CORS even for non-routable requests.
	b.cors.handle(conn)

	if routeData == nil {
		if conn.ctx.IsOptions() {
			return noExecuteArg, kit.ErrPreflight
		}

		return noExecuteArg, kit.ErrNoHandler
	}

	// Walk over all the query params
	conn.ctx.QueryArgs().VisitAll(
		func(key, value []byte) {
			params = append(
				params,
				httpmux.Param{
					Key:   utils.B2S(key),
					Value: utils.B2S(value),
				},
			)
		},
	)

	conn.ctx.PostArgs().VisitAll(
		func(key, value []byte) {
			params = append(
				params,
				httpmux.Param{
					Key:   utils.B2S(key),
					Value: utils.B2S(value),
				},
			)
		},
	)

	m, err := routeData.Decoder(params, in)
	if err != nil {
		return noExecuteArg, errors.Wrap(kit.ErrDecodeIncomingMessageFailed, err)
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

func (b *bundle) Start(_ context.Context, cfg kit.GatewayStartConfig) error {
	var (
		ln  net.Listener
		err error
	)
	if cfg.ReusePort {
		ln, err = reuseport.Listen("tcp4", b.listen)
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
	return b.srv.Shutdown()
}

func (b *bundle) Subscribe(d kit.GatewayDelegate) {
	b.d = d
}
