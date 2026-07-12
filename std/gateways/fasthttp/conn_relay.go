package fasthttp

import (
	"net/http"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp/proxy"

	"github.com/fasthttp/websocket"
)

var _ kit.RelayConn = (*httpConn)(nil)

func (c *httpConn) RequestBody() ([]byte, error) {
	return c.getBodyUncompressed()
}

func (c *httpConn) IsWebSocketUpgrade() bool {
	return websocket.FastHTTPIsWebSocketUpgrade(c.ctx)
}

func (c *httpConn) RelayHTTP(targetURL string, cfg kit.RelayConfig) error {
	body, err := c.RequestBody()
	if err != nil {
		return err
	}

	return proxy.RelayHTTP(c.ctx, c.GetMethod(), targetURL, body, cfg)
}

func (c *httpConn) RelayWebSocket(targetURL string, cfg kit.RelayConfig) error {
	return proxy.RelayWebSocket(c.ctx, targetURL, cfg)
}

func (c *httpConn) WriteHTTPResponse(status int, header http.Header, body []byte) error {
	c.ctx.Response.Reset()
	c.ctx.Response.SetStatusCode(status)

	for k, vs := range header {
		for _, v := range vs {
			c.ctx.Response.Header.Set(k, v)
		}
	}

	c.ctx.Response.SetBody(body)

	return kit.ErrRelayCompleted
}
