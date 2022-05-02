package storebased

import (
	"time"

	"github.com/clubpay/ronykit"
)

type Option func(cfg *config)

func WithID(id string) Option {
	return func(cfg *config) {
		cfg.id = id
	}
}

func WithHeartbeat(t time.Duration) Option {
	return func(cfg *config) {
		cfg.heartbeatInterval = t
	}
}

func WithStore(cs ronykit.ClusterStore) Option {
	return func(cfg *config) {
		cfg.store = cs
	}
}

func WithAdvertisedURL(urls ...string) Option {
	return func(cfg *config) {
		cfg.urls = urls
	}
}
