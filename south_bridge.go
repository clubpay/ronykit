package ronykit

import "sync"

type southBridge struct {
	ctxPool sync.Pool
	l       Logger
	c       Cluster
	eh      ErrHandler
}
