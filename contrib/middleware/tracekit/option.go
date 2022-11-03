package tracekit

import "github.com/clubpay/ronykit/kit"

type Option func(cfg *config)

type config struct {
    tracerName  string
    propagator  TracePropagator
    serviceName string
    env         string
    tags        map[string]string
    dynTags     func(ctx *kit.LimitedContext) map[string]string
}

func ServiceName(name string) Option {
    return func(cfg *config) {
        cfg.serviceName = name
    }
}

func Env(env string) Option {
    return func(cfg *config) {
        cfg.env = env
    }
}

func WithTags(tags map[string]string) Option {
    return func(cfg *config) {
        cfg.tags = tags
    }
}

func WithDynamicTags(f func(ctx *kit.LimitedContext) map[string]string) Option {
    return func(cfg *config) {
        cfg.dynTags = f
    }
}
