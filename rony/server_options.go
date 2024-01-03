package rony

import (
	"time"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

type serverConfig struct {
	services    map[string]*desc.Service
	edgeOpts    []kit.Option
	gatewayOpts []fasthttp.Option
}

func defaultServerConfig() serverConfig {
	return serverConfig{
		gatewayOpts: []fasthttp.Option{
			fasthttp.WithPredicateKey("cmd"),
		},
	}
}

func (cfg *serverConfig) getService(name string) *desc.Service {
	if cfg.services == nil {
		cfg.services = make(map[string]*desc.Service)
	}

	svc, ok := cfg.services[name]
	if !ok {
		svc = desc.NewService(name)
		cfg.services[name] = svc
	}

	return svc
}

func (cfg *serverConfig) allServices() []kit.ServiceDescriptor {
	svcs := make([]kit.ServiceDescriptor, 0, len(cfg.services))
	for _, svc := range cfg.services {
		svcs = append(svcs, svc)
	}

	return svcs
}

func (cfg serverConfig) Gateway() kit.Gateway {
	return fasthttp.MustNew(cfg.gatewayOpts...)
}

type ServerOption func(cfg *serverConfig)

type (
	CORSConfig = fasthttp.CORSConfig
)

func WithCORS(cors CORSConfig) ServerOption {
	return func(cfg *serverConfig) {
		cfg.gatewayOpts = append(cfg.gatewayOpts, fasthttp.WithCORS(cors))
	}
}

func Listen(addr string) ServerOption {
	return func(cfg *serverConfig) {
		cfg.gatewayOpts = append(cfg.gatewayOpts, fasthttp.Listen(addr))
	}
}

func WithServerName(name string) ServerOption {
	return func(cfg *serverConfig) {
		cfg.gatewayOpts = append(cfg.gatewayOpts, fasthttp.WithServerName(name))
	}
}

type CompressionLevel = fasthttp.CompressionLevel

// Represents compression level that will be used in the middleware
const (
	CompressionLevelDisabled        CompressionLevel = -1
	CompressionLevelDefault         CompressionLevel = 0
	CompressionLevelBestSpeed       CompressionLevel = 1
	CompressionLevelBestCompression CompressionLevel = 2
)

func WithCompression(lvl CompressionLevel) ServerOption {
	return func(cfg *serverConfig) {
		cfg.gatewayOpts = append(cfg.gatewayOpts, fasthttp.WithCompressionLevel(lvl))
	}
}

func WithPredicateKey(key string) ServerOption {
	return func(cfg *serverConfig) {
		cfg.gatewayOpts = append(cfg.gatewayOpts, fasthttp.WithPredicateKey(key))
	}
}

func WithWebsocketEndpoint(endpoint string) ServerOption {
	return func(cfg *serverConfig) {
		cfg.gatewayOpts = append(cfg.gatewayOpts, fasthttp.WithWebsocketEndpoint(endpoint))
	}
}

func WithCustomRPC(in kit.IncomingRPCFactory, out kit.OutgoingRPCFactory) ServerOption {
	return func(cfg *serverConfig) {
		cfg.gatewayOpts = append(cfg.gatewayOpts, fasthttp.WithCustomRPC(in, out))
	}
}

func WithTracer(tracer kit.Tracer) ServerOption {
	return func(cfg *serverConfig) {
		cfg.edgeOpts = append(cfg.edgeOpts, kit.WithTrace(tracer))
	}
}

func WithLogger(logger kit.Logger) ServerOption {
	return func(cfg *serverConfig) {
		cfg.edgeOpts = append(cfg.edgeOpts, kit.WithLogger(logger))
	}
}

func WithPrefork() ServerOption {
	return func(cfg *serverConfig) {
		cfg.edgeOpts = append(cfg.edgeOpts, kit.WithPrefork())
	}
}

func WithShutdownTimeout(timeout time.Duration) ServerOption {
	return func(cfg *serverConfig) {
		cfg.edgeOpts = append(cfg.edgeOpts, kit.WithShutdownTimeout(timeout))
	}
}

func WithGlobalHandlers(handlers ...kit.HandlerFunc) ServerOption {
	return func(cfg *serverConfig) {
		cfg.edgeOpts = append(cfg.edgeOpts, kit.WithGlobalHandlers(handlers...))
	}
}
