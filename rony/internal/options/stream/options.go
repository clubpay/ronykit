package stream

import (
	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/std/gateways/fasthttp"
)

type Config struct {
	Selectors []SelectorConfig
}

func GenConfig(opt ...Option) Config {
	cfg := Config{}

	for _, o := range opt {
		o(&cfg)
	}

	return cfg
}

type Option func(*Config)

type Selector struct {
	Name string
	S    kit.RPCRouteSelector
}

type SelectorConfig struct {
	Decoder  fasthttp.DecoderFunc
	Name     string
	Selector kit.RPCRouteSelector
}

type SelectorOption func(*SelectorConfig)

func GenSelectorConfig(opt ...SelectorOption) SelectorConfig {
	cfg := SelectorConfig{}

	for _, o := range opt {
		o(&cfg)
	}

	return cfg
}
