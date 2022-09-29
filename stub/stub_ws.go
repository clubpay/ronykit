package stub

import (
	"context"
	"io"

	"github.com/clubpay/ronykit/utils/reflector"
	"github.com/fasthttp/websocket"
)

type WSResponseHandler func(ctx context.Context, r WSEnvelope) *Error

type WSEnvelope interface {
	GetPredicate() string
	GetBody() []byte
	GetHeader(key string) string
}

type WebsocketCtx struct {
	cfg            wsConfig
	err            *Error
	handlers       map[string]WSResponseHandler
	defaultHandler WSResponseHandler
	r              *reflector.Reflector
	dumpReq        io.Writer
	dumpRes        io.Writer

	// fasthttp entities
	url string
	c   *websocket.Conn
}

func (wCtx *WebsocketCtx) Connect(ctx context.Context) error {
	var err error
	wCtx.c, _, err = wCtx.cfg.d.DialContext(ctx, wCtx.url, wCtx.cfg.upgradeHdr)

	return err
}
