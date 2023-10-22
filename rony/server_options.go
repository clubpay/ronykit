package rony

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/desc"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

type serverConfig struct {
	services map[string]*desc.Service
}

func defaultServerConfig() serverConfig {
	return serverConfig{}
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

type ServerOption func(cfg *serverConfig)

func (cfg serverConfig) Gateway() kit.Gateway {
	return fasthttp.MustNew(
		fasthttp.Listen(":8080"),
		fasthttp.WithCORS(fasthttp.CORSConfig{}),
	)
}
