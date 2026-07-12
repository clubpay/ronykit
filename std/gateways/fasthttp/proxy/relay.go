package proxy

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/clubpay/ronykit/kit"

	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// HopHeaders returns hop-by-hop header names stripped from relayed requests and responses.
func HopHeaders() []string {
	return append([]string(nil), hopHeaders...)
}

func RelayHTTP(
	ctx *fasthttp.RequestCtx,
	method string,
	targetURL string,
	body []byte,
	cfg kit.RelayConfig,
) error {
	u, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	if u.Scheme == "" || u.Host == "" {
		return errors.New("targetURL must be absolute")
	}

	u = dropQueryParams(u, cfg.DropQueryParams)

	req := fasthttp.AcquireRequest()
	res := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(res)

	copyRelayRequestHeaders(ctx, req, cfg)
	req.Header.SetMethod(method)
	setRequestURI(req, u)
	req.SetHost(u.Host)
	req.SetBody(body)

	if cfg.RewriteRequest != nil {
		view := &kit.RelayRequestView{
			Method: method,
			URL:    u,
			Header: requestHeaderToHTTP(&req.Header),
			Body:   append([]byte(nil), body...),
		}
		if err := cfg.RewriteRequest(view); err != nil {
			return err
		}

		applyRelayRequestView(req, view)
	}

	client := relayClientFor(cfg)
	if cfg.Timeout > 0 {
		err = client.DoTimeout(req, res, cfg.Timeout)
	} else {
		err = client.Do(req, res)
	}

	if err != nil {
		return err
	}

	for _, h := range hopHeaders {
		res.Header.Del(h)
	}

	statusCode := res.StatusCode()
	header := responseHeaderToHTTP(&res.Header)
	respBody := append([]byte(nil), res.Body()...)

	if cfg.RewriteResponse != nil {
		view := &kit.RelayResponseView{
			StatusCode: statusCode,
			Header:     header,
			Body:       respBody,
		}
		if err := cfg.RewriteResponse(view); err != nil {
			return err
		}

		statusCode = view.StatusCode
		header = view.Header
		respBody = view.Body
	}

	ctx.Response.Reset()
	ctx.Response.SetStatusCode(statusCode)
	copyHTTPHeaderToResponseHeader(&ctx.Response.Header, header)
	ctx.Response.SetBody(respBody)

	return kit.ErrRelayCompleted
}

func RelayWebSocket(ctx *fasthttp.RequestCtx, targetURL string, cfg kit.RelayConfig) error {
	u, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	if u.Scheme != "ws" && u.Scheme != "wss" {
		return errors.New("targetURL must use ws:// or wss://")
	}

	u = dropQueryParams(u, cfg.DropQueryParams)

	dialer := DefaultDialer
	if cfg.TLSConfig != nil {
		dialer = &websocket.Dialer{TLSClientConfig: cfg.TLSConfig}
	}

	upgrader := cloneWSUpgrader(cfg)

	forwardHeader, err := buildRelayWebSocketDialHeaders(ctx, u, cfg)
	if err != nil {
		return err
	}

	connBackend, respBackend, err := dialer.Dial(u.String(), forwardHeader)
	if err != nil {
		if respBackend != nil {
			_ = wsCopyResponse(&ctx.Response, respBackend)
		} else {
			ctx.Error(err.Error(), fasthttp.StatusServiceUnavailable)
		}

		return err
	}

	err = upgrader.Upgrade(ctx, func(connPub *websocket.Conn) {
		defer connPub.Close()
		defer connBackend.Close()

		errClient := make(chan error, 1)
		errBackend := make(chan error, 1)

		go replicateWebsocketConn(&nopLogger{}, connPub, connBackend, errClient)
		go replicateWebsocketConn(&nopLogger{}, connBackend, connPub, errBackend)

		// Wait for either side to finish; the deferred Close calls unblock the other
		// replicate goroutine so it can exit as well.
		select {
		case <-errClient:
		case <-errBackend:
		}
	})
	if err != nil {
		// Upgrade failed before Hijack scheduled the handler (e.g. rejected origin),
		// so the deferred connBackend.Close in the handler never runs. Close it here
		// to avoid leaking the already-dialed upstream connection.
		_ = connBackend.Close()

		return err
	}

	return kit.ErrRelayCompleted
}

// wsDialStripHeaders removes WebSocket handshake headers that must not be forwarded
// to websocket.Dialer; the dialer generates its own values for these fields.
var wsDialStripHeaders = []string{
	"Sec-Websocket-Key",
	"Sec-Websocket-Version",
	"Sec-Websocket-Extensions",
}

func buildRelayWebSocketDialHeaders(
	ctx *fasthttp.RequestCtx,
	u *url.URL,
	cfg kit.RelayConfig,
) (http.Header, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	copyRelayRequestHeaders(ctx, req, cfg)
	req.Header.SetMethod(http.MethodGet)
	setRequestURI(req, u)
	req.SetHost(u.Host)

	if cfg.RewriteRequest != nil {
		view := &kit.RelayRequestView{
			Method: http.MethodGet,
			URL:    u,
			Header: requestHeaderToHTTP(&req.Header),
		}
		if err := cfg.RewriteRequest(view); err != nil {
			return nil, err
		}

		applyRelayRequestView(req, view)
	}

	hdr := requestHeaderToHTTP(&req.Header)
	for _, h := range wsDialStripHeaders {
		hdr.Del(h)
	}

	return hdr, nil
}

func copyRelayRequestHeaders(ctx *fasthttp.RequestCtx, req *fasthttp.Request, cfg kit.RelayConfig) {
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		k := string(key)
		if shouldDropHeader(k, cfg.DropRequestHeaders) {
			return
		}

		req.Header.Set(k, string(value))
	})

	for _, h := range hopHeaders {
		req.Header.Del(h)
	}

	for k, v := range cfg.ExtraRequestHeaders {
		req.Header.Set(k, v)
	}

	ip, _, err := net.SplitHostPort(ctx.RemoteAddr().String())
	if err == nil {
		if prior := req.Header.Peek(fasthttp.HeaderXForwardedFor); len(prior) > 0 {
			ip = string(prior) + ", " + ip
		}

		req.Header.Set(fasthttp.HeaderXForwardedFor, ip)
	}
}

func shouldDropHeader(name string, extra []string) bool {
	name = http.CanonicalHeaderKey(name)
	for _, h := range hopHeaders {
		if http.CanonicalHeaderKey(h) == name {
			return true
		}
	}

	for _, h := range extra {
		if http.CanonicalHeaderKey(h) == name {
			return true
		}
	}

	return false
}

func dropQueryParams(u *url.URL, drop []string) *url.URL {
	if len(drop) == 0 {
		return u
	}

	out := *u
	q := out.Query()

	for _, key := range drop {
		q.Del(key)
	}

	out.RawQuery = q.Encode()

	return &out
}

func setRequestURI(req *fasthttp.Request, u *url.URL) {
	uri := fasthttp.AcquireURI()
	defer fasthttp.ReleaseURI(uri)

	uri.SetScheme(u.Scheme)
	uri.SetHost(u.Host)
	uri.SetPath(u.Path)
	uri.SetQueryString(u.RawQuery)
	req.SetURI(uri)
}

func applyRelayRequestView(req *fasthttp.Request, view *kit.RelayRequestView) {
	// Reset first, then re-apply from the view so the method and URI set below are
	// not wiped by Reset (which resets method to GET and clears the request URI).
	req.Header.Reset()
	copyHTTPHeaderToRequestHeader(&req.Header, view.Header)

	if view.Method != "" {
		req.Header.SetMethod(view.Method)
	}

	if view.URL != nil {
		setRequestURI(req, view.URL)
		req.SetHost(view.URL.Host)
	}

	req.SetBody(view.Body)
}

func requestHeaderToHTTP(hdr *fasthttp.RequestHeader) http.Header {
	out := make(http.Header)

	hdr.VisitAll(func(key, value []byte) {
		out.Add(string(key), string(value))
	})

	return out
}

func responseHeaderToHTTP(hdr *fasthttp.ResponseHeader) http.Header {
	out := make(http.Header)

	hdr.VisitAll(func(key, value []byte) {
		out.Add(string(key), string(value))
	})

	return out
}

func copyHTTPHeaderToRequestHeader(dst *fasthttp.RequestHeader, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Set(k, v)
		}
	}
}

func copyHTTPHeaderToResponseHeader(dst *fasthttp.ResponseHeader, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Set(k, v)
		}
	}
}

func cloneWSUpgrader(cfg kit.RelayConfig) *websocket.FastHTTPUpgrader {
	if len(cfg.WebSocketSubprotocols) == 0 && cfg.WebSocketCheckOrigin == nil {
		return DefaultUpgrader
	}

	upgrader := &websocket.FastHTTPUpgrader{
		ReadBufferSize:  DefaultUpgrader.ReadBufferSize,
		WriteBufferSize: DefaultUpgrader.WriteBufferSize,
	}
	if len(cfg.WebSocketSubprotocols) > 0 {
		upgrader.Subprotocols = append([]string(nil), cfg.WebSocketSubprotocols...)
	}

	if cfg.WebSocketCheckOrigin != nil {
		check := cfg.WebSocketCheckOrigin
		upgrader.CheckOrigin = func(ctx *fasthttp.RequestCtx) bool {
			return check(string(ctx.Request.Header.Peek(fasthttp.HeaderOrigin)))
		}
	}

	return upgrader
}

// relayClientNoTLS handles all relays without a custom TLS config. fasthttp.Client
// lazily creates a per-host HostClient and runs a cleaner goroutine that evicts
// idle hosts (no open conns, no pending requests), so memory stays bounded even
// when relaying to a large or churning set of upstream hosts.
var relayClientNoTLS = &fasthttp.Client{Name: _fasthttpHostClientName}

// relayClientsByTLS caches one fasthttp.Client per *tls.Config pointer. Apps
// typically reuse a single tls.Config, so this map is bounded by the number of
// distinct configs (usually one), and each client self-evicts idle hosts.
var relayClientsByTLS sync.Map // *tls.Config -> *fasthttp.Client

func relayClientFor(cfg kit.RelayConfig) *fasthttp.Client {
	if cfg.TLSConfig == nil {
		return relayClientNoTLS
	}

	if v, ok := relayClientsByTLS.Load(cfg.TLSConfig); ok {
		return v.(*fasthttp.Client) //nolint:forcetypeassert
	}

	client := &fasthttp.Client{
		Name:      _fasthttpHostClientName,
		TLSConfig: cfg.TLSConfig,
	}
	actual, _ := relayClientsByTLS.LoadOrStore(cfg.TLSConfig, client)

	return actual.(*fasthttp.Client) //nolint:forcetypeassert
}
