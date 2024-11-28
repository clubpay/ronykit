package kit

import (
	"context"
	"time"
)

// Store identifies a key-value store, which its value doesn't have any limitation.
type Store interface {
	Get(key string) any
	Exists(key string) bool
	Set(key string, val any)
	Delete(key string)
	Scan(prefix string, cb func(key string) bool)
}

// ClusterStore identifies a key-value store, which its key-value pairs are shared between
// different instances of the cluster. Also, the value type is only string.
type ClusterStore interface {
	// Set creates/updates a key-value pair in the cluster
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	SetMulti(ctx context.Context, kv map[string]string, ttl time.Duration) error
	// Delete deletes the key-value pair from the cluster
	Delete(ctx context.Context, key string) error
	// Get returns the value bind to the key
	Get(ctx context.Context, key string) (string, error)
	// Scan scans through the keys which have the prefix.
	// If callback returns `false`, then the scan is aborted.
	Scan(ctx context.Context, prefix string, cb func(string) bool) error
	ScanWithValue(ctx context.Context, prefix string, cb func(string, string) bool) error
}
