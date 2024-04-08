package fasthttp

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net"
	"sync"
	"sync/atomic"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/common"
	"github.com/clubpay/ronykit/kit/errors"
	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/kit/utils/buf"
	"github.com/clubpay/ronykit/std/gateways/fasthttp/internal/realip"
	"github.com/clubpay/ronykit/std/gateways/fasthttp/proxy"
	"github.com/fasthttp/router"
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
	utils    util

	reverseProxyPath string
	reverseProxy     *proxy.ReverseProxy
	httpRouter       *router.Router
	compress         CompressionLevel

	wsUpgrade     websocket.FastHTTPUpgrader
	wsRoutes      map[string]*routeData
	wsEndpoint    string
	wsNextID      uint64
	predicateKey  string
	rpcInFactory  kit.IncomingRPCFactory
	rpcOutFactory kit.OutgoingRPCFactory
}

var _ kit.Gateway = (*bundle)(nil)

func New(opts ...Option) (kit.Gateway, error) {
	r := &bundle{
		httpRouter:    router.New(),
		compress:      CompressionLevelDefault,
		wsRoutes:      map[string]*routeData{},
		srv:           &fasthttp.Server{},
		rpcInFactory:  common.SimpleIncomingJSONRPC,
		rpcOutFactory: common.SimpleOutgoingJSONRPC,
		l:             common.NewNopLogger(),
		utils:         defaultUtil(),
	}

	r.wsUpgrade.CheckOrigin = func(ctx *fasthttp.RequestCtx) bool {
		return true
	}
	for _, opt := range opts {
		opt(r)
	}

	r.httpRouter.HandleOPTIONS = true
	r.httpRouter.GlobalOPTIONS = r.cors.handle
	httpHandler := r.httpRouter.Handler
	switch r.compress {
	case CompressionLevelDefault:
		httpHandler = fasthttp.CompressHandlerBrotliLevel(
			r.httpRouter.Handler,
			fasthttp.CompressBrotliDefaultCompression,
			fasthttp.CompressDefaultCompression,
		)
	case CompressionLevelBestSpeed:
		httpHandler = fasthttp.CompressHandlerBrotliLevel(
			r.httpRouter.Handler,
			fasthttp.CompressBrotliBestSpeed,
			fasthttp.CompressBestSpeed,
		)
	case CompressionLevelBestCompression:
		httpHandler = fasthttp.CompressHandlerBrotliLevel(
			r.httpRouter.Handler,
			fasthttp.CompressBrotliBestCompression,
			fasthttp.CompressBestCompression,
		)
	}

	if r.reverseProxy != nil {
		r.httpRouter.ANY(r.reverseProxyPath, r.reverseProxy.ServeHTTP)
	}

	if r.wsEndpoint != "" {
		r.httpRouter.GET(r.wsEndpoint, r.wsHandler)
	}

	r.srv.Handler = httpHandler

	return r, nil
}

func MustNew(opts ...Option) kit.Gateway {
	b, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return b
}

type routeData struct {
	Method      string
	Path        string
	Predicate   string
	ServiceName string
	ContractID  string
	Decoder     DecoderFunc
	Factory     kit.MessageFactoryFunc
}

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

	rd := &routeData{
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
		panic("REST selector MUST have method and path")
	}

	decoder, ok := restSelector.Query(queryDecoder).(DecoderFunc)
	if !ok || decoder == nil {
		decoder = reflectDecoder(enc, kit.CreateMessageFactory(input))
	}

	b.httpRouter.Handle(
		restSelector.GetMethod(), restSelector.GetPath(),
		b.genHTTPHandler(
			routeData{
				ServiceName: svcName,
				ContractID:  contractID,
				Method:      restSelector.GetMethod(),
				Path:        restSelector.GetPath(),
				Decoder:     decoder,
			},
		),
	)
}

func (b *bundle) genHTTPHandler(rd routeData) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		c, ok := b.connPool.Get().(*httpConn)
		if !ok {
			c = &httpConn{}
		}

		c.ctx = ctx
		c.rd = &rd
		b.d.OnOpen(c)
		b.d.OnMessage(c, b.writeFunc, ctx.PostBody())
		b.d.OnClose(c.ConnID())

		b.connPool.Put(c)
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
	_ = b.wsUpgrade.Upgrade(
		ctx,
		func(conn *websocket.Conn) {
			wsc := &wsConn{
				kv:       map[string]string{},
				id:       atomic.AddUint64(&b.wsNextID, 1),
				clientIP: realip.FromRequest(ctx),
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
	switch v := msg.(type) {
	case kit.MultipartFormMessage:
		x := kit.RawMessage{}
		err = inputMsgContainer.ExtractMessage(&x)
		if err != nil {
			return noExecuteArg, errors.Wrap(kit.ErrDecodeIncomingMessageFailed, err)
		}

		frm, err := multipart.NewReader(
			bytes.NewReader(x),
			utils.B2S(getMultipartFormBoundary(utils.S2B(inputMsgContainer.GetHdr("Content-Type")))),
		).ReadForm(int64(b.srv.MaxRequestBodySize))
		if err != nil {
			return noExecuteArg, errors.Wrap(kit.ErrDecodeIncomingMessageFailed, err)
		}

		v.SetForm(frm)
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

func (b *bundle) httpDispatch(ctx *kit.Context, in []byte) (kit.ExecuteArg, error) {
	//nolint:forcetypeassert
	conn := ctx.Conn().(*httpConn)

	// Check CORS rules before even returning errRouteNotFound.
	// This makes sure that we handle any CORS even for non-routable requests.
	b.cors.handle(conn.ctx)

	if conn.rd == nil {
		if conn.ctx.IsOptions() {
			return noExecuteArg, kit.ErrPreflight
		}

		return noExecuteArg, kit.ErrNoHandler
	}

	m, err := conn.rd.Decoder(conn.ctx, in)
	if err != nil {
		return noExecuteArg, errors.Wrap(kit.ErrDecodeIncomingMessageFailed, err)
	}

	ctx.In().
		SetHdrWalker(conn).
		SetMsg(m)

	return kit.ExecuteArg{
		ServiceName: conn.rd.ServiceName,
		ContractID:  conn.rd.ContractID,
		Route:       fmt.Sprintf("%s %s", conn.rd.Method, conn.rd.Path),
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

var (
	strMultipartFormData = []byte("multipart/form-data")
	strBoundary          = []byte("boundary")
)

func getMultipartFormBoundary(contentType []byte) []byte {
	b := contentType
	if !bytes.HasPrefix(b, strMultipartFormData) {
		return nil
	}
	b = b[len(strMultipartFormData):]
	if len(b) == 0 || b[0] != ';' {
		return nil
	}

	var n int
	for len(b) > 0 {
		n++
		for len(b) > n && b[n] == ' ' {
			n++
		}
		b = b[n:]
		if !bytes.HasPrefix(b, strBoundary) {
			if n = bytes.IndexByte(b, ';'); n < 0 {
				return nil
			}
			continue
		}

		b = b[len(strBoundary):]
		if len(b) == 0 || b[0] != '=' {
			return nil
		}
		b = b[1:]
		if n = bytes.IndexByte(b, ';'); n >= 0 {
			b = b[:n]
		}
		if len(b) > 1 && b[0] == '"' && b[len(b)-1] == '"' {
			b = b[1 : len(b)-1]
		}
		return b
	}
	return nil
}
