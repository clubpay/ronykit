package settings

import "github.com/spf13/viper"

type (
	Option         = viper.Option
	StringReplacer = viper.StringReplacer
)

var (
	KeyDelimiter   = viper.KeyDelimiter
	EnvKeyReplacer = viper.EnvKeyReplacer
)
