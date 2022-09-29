package stub

import (
	"net/http"

	"github.com/fasthttp/websocket"
)

type WebsocketOption func(cfg *wsConfig)

type wsConfig struct {
	upgradeHdr http.Header
	d          *websocket.Dialer
}

func WithUpgradeHeader(key string, values ...string) WebsocketOption {
	return func(cfg *wsConfig) {
		if cfg.upgradeHdr == nil {
			cfg.upgradeHdr = http.Header{}
		}
		cfg.upgradeHdr[key] = values
	}
}

func WithCustomDialer(d *websocket.Dialer) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.d = d
	}
}
