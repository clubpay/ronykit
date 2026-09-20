---
import_path: github.com/clubpay/ronykit/x/settings
short_name: settings
---

Viper-backed configuration with env/file/flags/defaults priority and struct unmarshaling via the `settings` tag. Use this instead of
`os.Getenv`, `flag`, or raw `spf13/viper`.

## Usage Hint

`settings.New()` returns a **value**. `SetFromFile` and `Unmarshal` are methods on `*Settings`.

```go
func New(set settings.Settings) (*Settings, error) {
	_ = set.SetFromFile(ConfigName, ConfigPaths...)
	mod := &Settings{}
	if err := set.Unmarshal(mod); err != nil {
		return nil, err
	}
	return mod, nil
}

type Settings struct {
	DB DBConfig `settings:"db"`
}
type DBConfig struct {
	Host string `settings:"host"`
	Port int    `settings:"port"`
}
```

The workspace injects `settings.Settings` via fx; feature `New` unmarshals into the module struct. Do not call `settings.Unmarshal` as a
package function — it does not exist.
