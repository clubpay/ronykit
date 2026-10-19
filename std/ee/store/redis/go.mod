module github.com/clubpay/ronykit/std/ee/store/redis

go 1.27.0

require (
	github.com/clubpay/ronykit/ee v0.0.2
	github.com/clubpay/ronykit/x/rkit v0.6.3
	github.com/redis/go-redis/v9 v9.23.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/goccy/go-reflect v1.2.0 // indirect
	github.com/jinzhu/copier v0.4.0 // indirect
	go.uber.org/atomic v1.12.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// ee is not published yet; resolve it from this repository.
replace github.com/clubpay/ronykit/ee => ../../../../ee
