package stub

import (
	"net/http"
	"sync"
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/fasthttp/websocket"
)

type Dialer = websocket.Dialer

type PreDialHandler func(d *Dialer)

type OnConnectHandler func(ctx *WebsocketCtx)

type WebsocketOption func(cfg *wsConfig)

type wsConfig struct {
	predicateKey    string
	rpcInFactory    kit.IncomingRPCFactory
	rpcOutFactory   kit.OutgoingRPCFactory
	ratelimitChan   chan struct{}
	handlersWG      sync.WaitGroup
	handlers        map[string]RPCContainerHandler
	defaultHandler  RPCContainerHandler
	tracePropagator kit.TracePropagator

	autoReconnect bool
	dialerBuilder func() *websocket.Dialer
	upgradeHdr    http.Header
	pingTime      time.Duration
	dialTimeout   time.Duration
	writeTimeout  time.Duration

	preDial    func(d *websocket.Dialer)
	onConnect  OnConnectHandler
	preflights []RPCPreflightHandler

	panicRecoverFunc func(err any)
}

func WithUpgradeHeader(key string, values ...string) WebsocketOption {
	return func(cfg *wsConfig) {
		if cfg.upgradeHdr == nil {
			cfg.upgradeHdr = http.Header{}
		}
		cfg.upgradeHdr[key] = values
	}
}

func WithCustomDialerBuilder(b func() *websocket.Dialer) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.dialerBuilder = b
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

func WithCustomRPC(in kit.IncomingRPCFactory, out kit.OutgoingRPCFactory) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.rpcInFactory = in
		cfg.rpcOutFactory = out
	}
}

func WithOnConnectHandler(f OnConnectHandler) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.onConnect = f
	}
}

func WithPreDialHandler(f PreDialHandler) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.preDial = f
	}
}

func WithPredicateKey(key string) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.predicateKey = key
	}
}

func WithAutoReconnect(b bool) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.autoReconnect = b
	}
}

func WithPingTime(t time.Duration) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.pingTime = t
	}
}

func WithConcurrency(n int) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.ratelimitChan = make(chan struct{}, n)
	}
}

func WithRecoverPanic(f func(err any)) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.panicRecoverFunc = f
	}
}

// WithPreflightRPC register one or many handlers to run in sequence before
// actually making requests.
func WithPreflightRPC(h ...RPCPreflightHandler) WebsocketOption {
	return func(cfg *wsConfig) {
		cfg.preflights = append(cfg.preflights[:0], h...)
	}
}
