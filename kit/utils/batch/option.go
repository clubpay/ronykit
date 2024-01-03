package batch

import (
	"runtime"
	"time"
)

type config struct {
	maxWorkers  int32
	batchSize   int32
	minWaitTime time.Duration
}

var defaultConfig = config{
	maxWorkers:  int32(runtime.NumCPU() * 10),
	batchSize:   100,
	minWaitTime: 0,
}

type Option func(*config)

// WithMaxWorkers sets the maximum number of workers to use.
// Defaults to runtime.NumCPU() * 10.
func WithMaxWorkers(maxWorkers int32) Option {
	return func(c *config) {
		c.maxWorkers = maxWorkers
	}
}

// WithBatchSize sets the maximum number of entries to batch together.
// Defaults to 100.
func WithBatchSize(batchSize int32) Option {
	return func(c *config) {
		c.batchSize = batchSize
	}
}

// WithMinWaitTime sets the minimum amount of time to wait before flushing
// a batch. Defaults to 0.
func WithMinWaitTime(minWaitTime time.Duration) Option {
	return func(c *config) {
		c.minWaitTime = minWaitTime
	}
}
