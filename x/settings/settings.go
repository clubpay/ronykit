package settings

import (
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// Settings is a prioritized configuration registry. It
// maintains a set of configuration sources, fetches
// values to populate those, and provides them according
// to the source's priority.
// The priority of the sources is the following:
// 1. overrides
// 2. flags
// 3. env. variables
// 4. config file
// 5. key/value store
// 6. defaults
type Settings struct {
	v *viper.Viper
}

func New() Settings {
	v := viper.NewWithOptions(
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)
	v.SetEnvPrefix("")
	v.AutomaticEnv()

	return Settings{v: v}
}

func (s *Settings) GetString(key string) string {
	return s.v.GetString(key)
}

func (s *Settings) GetInt(key string) int {
	return s.v.GetInt(key)
}

func (s *Settings) GetBool(key string) bool {
	return s.v.GetBool(key)
}

func (s *Settings) GetDuration(key string) time.Duration {
	return s.v.GetDuration(key)
}

func (s *Settings) GetStringSlice(key string) []string {
	return s.v.GetStringSlice(key)
}

func (s *Settings) GetStringMap(key string) map[string]any {
	return s.v.GetStringMap(key)
}

func (s *Settings) GetStringMapString(key string) map[string]string {
	return s.v.GetStringMapString(key)
}

func (s *Settings) GetStringMapStringSlice(key string) map[string][]string {
	return s.v.GetStringMapStringSlice(key)
}

func (s *Settings) GetTime(key string) time.Time {
	return s.v.GetTime(key)
}

func (s *Settings) GetFloat64(key string) float64 {
	return s.v.GetFloat64(key)
}

func (s *Settings) GetInt64(key string) int64 {
	return s.v.GetInt64(key)
}

func (s *Settings) GetUint64(key string) uint64 {
	return s.v.GetUint64(key)
}

func (s *Settings) GetUint(key string) uint {
	return s.v.GetUint(key)
}

func (s *Settings) GetInt32(key string) int32 {
	return s.v.GetInt32(key)
}

func (s *Settings) GetUint32(key string) uint32 {
	return s.v.GetUint32(key)
}

func (s *Settings) Get(key string) any {
	return s.v.Get(key)
}

func (s *Settings) Set(key string, value any) {
	s.v.Set(key, value)
}

func (s *Settings) SetDefault(key string, value any) {
	s.v.SetDefault(key, value)
}

func (s *Settings) AllKeys() []string {
	return s.v.AllKeys()
}

func (s *Settings) AllSettings() map[string]any {
	return s.v.AllSettings()
}

// AddConfigPath adds a path for Viper to search for the config file in.
// Can be called multiple times to define multiple search paths.
func (s *Settings) AddConfigPath(path string) {
	s.v.AddConfigPath(path)
}

// SetConfigName sets name for the config file.
// Does not include extension.
func (s *Settings) SetConfigName(name string) {
	s.v.SetConfigName(name)
}

// ReadInConfig will discover and load the configuration file from disk
// and key/value stores, searching in one of the defined paths.
func (s *Settings) ReadInConfig() error {
	return s.v.ReadInConfig()
}

// WatchConfig starts watching a config file for changes.
func (s *Settings) WatchConfig() {
	s.v.WatchConfig()
}

// WriteConfig writes the current configuration to a given filename.
func (s *Settings) WriteConfig(filename string) error {
	return s.v.WriteConfigAs(filename)
}

func (s *Settings) SetFromFile(configFileName string, configSearchPaths ...string) error {
	for _, p := range configSearchPaths {
		s.v.AddConfigPath(p)
	}

	if configFileName == "" {
		configFileName = "config"
	}

	s.v.SetConfigName(configFileName)

	return s.v.ReadInConfig()
}

// AutomaticEnv makes Settings check if environment variables match any of the existing keys
// (config, default or flags). If matching env vars are found, they are loaded into Viper.
func (s *Settings) AutomaticEnv() {
	s.v.AutomaticEnv()
}

func (s *Settings) Unmarshal(v any) error {
	c := &mapstructure.DecoderConfig{
		Metadata:             nil,
		Result:               v,
		TagName:              "settings",
		IgnoreUntaggedFields: true,
		WeaklyTypedInput:     true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	}

	decoder, err := mapstructure.NewDecoder(c)
	if err != nil {
		return err
	}

	return decoder.Decode(s.AllSettings())
}
