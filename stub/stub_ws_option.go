package stub

import (
	"net/http"

	"github.com/clubpay/ronykit"
	"github.com/fasthttp/websocket"
)

type WebsocketOption func(cfg *wsConfig)

type wsConfig struct {
	upgradeHdr     http.Header
	predicateKey   string
	d              *websocket.Dialer
	rpcInFactory   ronykit.IncomingRPCFactory
	rpcOutFactory  ronykit.OutgoingRPCFactory
	handlers       map[string]RPCContainerHandler
	defaultHandler RPCContainerHandler
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

func WithDefaultHandler(h RPCContainerHandler) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.defaultHandler = h
	}
}

func WithHandler(predicate string, h RPCContainerHandler) WebsocketOption {
	return func(cfg *wsConfig) {
		if cfg.handlers == nil {
			cfg.handlers = map[string]RPCContainerHandler{}
		}
		cfg.handlers[predicate] = h
	}
}

func WithCustomRPC(in ronykit.IncomingRPCFactory, out ronykit.OutgoingRPCFactory) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.rpcInFactory = in
		cfg.rpcOutFactory = out
	}
}

func WithPredicateKey(key string) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.predicateKey = key
	}
}
