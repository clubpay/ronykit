---
import_path: github.com/clubpay/ronykit/x/settings
short_name: settings
---
Viper-backed configuration with env/file/flags/defaults priority and struct unmarshaling via `settings` tag.

## Usage Hint

Define a typed settings struct, call settings.Unmarshal, and inject via fx into api/app layers.
