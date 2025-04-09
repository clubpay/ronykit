package settings

import "time"

// Settings defines the configurable parameters of the system.
// Implementations: viper.Settings
type Settings interface {
	SetDefault(key string, val interface{})
	GetString(key string) string
	GetDuration(key string) time.Duration
	GetStringSlice(key string) []string
	GetInt(key string) int
	GetBool(key string) bool
	GetInt64(key string) int64
}
